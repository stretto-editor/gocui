// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocui

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/nsf/termbox-go"
)

type geom interface {
	Size() (x, y int)
	Name() string
	Position() (x0, y0, x1, y1 int)
	draw() error
}

// Container is a geometry element to create Views hierarchy
type Container struct {
	name           string
	x0, y0, x1, y1 int
	childrens      []geom
}

// newView returns a new View object.
func newViewNode(name string, x0, y0, x1, y1 int) *Container {
	c := &Container{
		name: name,
		x0:   x0,
		y0:   y0,
		x1:   x1,
		y1:   y1,
	}
	return c
}

// Size returns the number of visible columns and rows in the container.
func (c *Container) Size() (x, y int) {
	return c.x1 - c.x0 - 1, c.y1 - c.y0 - 1
}

// Name returns the name of the view.s
func (c *Container) Name() string {
	return c.name
}

// Children returns a slice of children
func (c *Container) children() []geom {
	return c.childrens
}

// Position returns coordinates
func (c *Container) Position() (int, int, int, int) {
	return c.x0, c.y0, c.x1, c.y1
}

func (c *Container) draw() error {
	for _, g := range c.childrens {
		g.draw()
	}
	return nil
}

// RoundRobinForward takes the last element in c.childrens and put it
// at the beginning of the same slice
func (c *Container) RoundRobinForward() *View {
	if len(c.childrens) <= 1 {
		return nil
	}
	c.childrens = append(c.childrens[1:], c.childrens[0])
	if v, ok := c.childrens[len(c.childrens)-1].(*View); ok {
		return v
	}
	return nil
}

// RoundRobinBackward takes the first element in c.childrens and put it
// at the end of the same slice
func (c *Container) RoundRobinBackward() *View {
	if len(c.childrens) <= 1 {
		return nil
	}
	c.childrens = append(c.childrens[len(c.childrens)-1:len(c.childrens)], c.childrens[:len(c.childrens)-1]...)
	return c.LastView()
}

// HasNoChildren checks the length of childrens
func (c *Container) HasNoChildren() bool {
	return len(c.childrens) == 0
}

func (c *Container) LastView() *View {
	for i := len(c.childrens) - 1; i >= 0; i-- {
		if v, ok := c.childrens[i].(*View); ok {
			return v
		}
	}
	return nil
}

// A View is a window. It maintains its own internal buffer and cursor
// position.
type View struct {
	name           string
	x0, y0, x1, y1 int
	ox, oy         int
	cx, cy         int
	lines          [][]rune
	readOffset     int
	readCache      string
	searchString   string

	Actions Context

	tainted   bool       // marks if the viewBuffer must be updated
	viewLines []viewLine // internal representation of the view's buffer

	Hidden bool // if true the view will not be drawn

	// BgColor and FgColor allow to configure the background and foreground
	// colors of the View.
	BgColor, FgColor Attribute

	// SelBgColor and SelFgColor are used to configure the background and
	// foreground colors of the selected line, when it is highlighted.
	SelBgColor, SelFgColor Attribute

	// If Editable is true, keystrokes will be added to the view's internal
	// buffer at the cursor position.
	Editable bool

	// Overwrite enables or disables the overwrite mode of the view.
	Overwrite bool

	// If Highlight is true, Sel{Bg,Fg}Colors will be used
	// for the line under the cursor position.
	Highlight bool

	// If Frame is true, a border will be drawn around the view.
	Frame bool

	// If Wrap is true, the content that is written to this View is
	// automatically wrapped when it is longer than its width. If true the
	// view's x-origin will be ignored.
	Wrap bool

	// If Autoscroll is true, the View will automatically scroll down when the
	// text overflows. If true the view's y-origin will be ignored.
	Autoscroll bool

	// If Frame is true, Title allows to configure a title for the view.
	// Title will be placed on the top-left corner
	Title string

	// If Frame is true, Footer allows to configure a footer for the view.
	// Title will be placed on the bottom-right corner
	Footer string

	// If Mask is true, the View will display the mask instead of the real
	// content
	Mask rune
}

type viewLine struct {
	linesX, linesY int // coordinates relative to v.lines
	line           []rune
}

// newView returns a new View object.
func newView(name string, x0, y0, x1, y1 int) *View {
	v := &View{
		name:    name,
		x0:      x0,
		y0:      y0,
		x1:      x1,
		y1:      y1,
		Frame:   true,
		tainted: true,
	}
	return v
}

func (v *View) GetSearchString() string {
	return v.searchString
}

// Size returns the number of visible columns and rows in the View.
func (v *View) Size() (x, y int) {
	return v.x1 - v.x0 - 1, v.y1 - v.y0 - 1
}

// Name returns the name of the view.
func (v *View) Name() string {
	return v.name
}

// Position returns coordinates
func (v *View) Position() (int, int, int, int) {
	return v.x0, v.y0, v.x1, v.y1
}

// Size returns the number of lines contained in the buffer
func (v *View) BufferSize() int {
	return len(v.lines)
}

// setRune writes a rune at the given point, relative to the view. It
// checks if the position is valid and applies the view's colors, taking
// into account if the cell must be highlighted.
func (v *View) setRune(x, y int, ch rune) error {
	maxX, maxY := v.Size()
	if x < 0 || x >= maxX || y < 0 || y >= maxY {
		return errors.New("invalid point")
	}

	var (
		ry, rcy int
		err     error
	)
	if v.Highlight {
		_, ry, err = v.realPosition(x, y)
		if err != nil {
			return err
		}
		_, rcy, err = v.realPosition(v.cx, v.cy)
		if err != nil {
			return err
		}
	}

	var fgColor, bgColor Attribute
	if v.Highlight && ry == rcy {
		fgColor = v.SelFgColor
		bgColor = v.SelBgColor
	} else {
		fgColor = v.FgColor
		bgColor = v.BgColor
	}

	if v.Mask != 0 {
		ch = v.Mask
	}
	termbox.SetCell(v.x0+x+1, v.y0+y+1, ch,
		termbox.Attribute(fgColor), termbox.Attribute(bgColor))
	return nil
}

// SetCursor sets the cursor position of the view at the given point,
// relative to the view. It checks if the position is valid.
func (v *View) SetCursor(x, y int) error {
	maxX, maxY := v.Size()
	if x < 0 || x >= maxX || y < 0 || y >= maxY {
		return errors.New("invalid point")
	}
	v.cx = x
	v.cy = y
	return nil
}

// Cursor returns the cursor position of the view.
func (v *View) Cursor() (x, y int) {
	return v.cx, v.cy
}

// SetOrigin sets the origin position of the view's internal buffer,
// so the buffer starts to be printed from this point, which means that
// it is linked with the origin point of view. It can be used to
// implement Horizontal and Vertical scrolling with just incrementing
// or decrementing ox and oy.
func (v *View) SetOrigin(x, y int) error {
	if x < 0 || y < 0 {
		return errors.New("invalid point")
	}
	v.ox = x
	v.oy = y
	return nil
}

// Origin returns the origin position of the view.
func (v *View) Origin() (x, y int) {
	return v.ox, v.oy
}

// Write appends a byte slice into the view's internal buffer. Because
// View implements the io.Writer interface, it can be passed as parameter
// of functions like fmt.Fprintf, fmt.Fprintln, io.Copy, etc. Clear must
// be called to clear the view's buffer.
func (v *View) Write(p []byte) (n int, err error) {
	v.tainted = true

	for _, ch := range bytes.Runes(p) {
		switch ch {
		case '\n':
			v.lines = append(v.lines, nil)
		case '\r':
			nl := len(v.lines)
			if nl > 0 {
				v.lines[nl-1] = nil
			} else {
				v.lines = make([][]rune, 1)
			}
		default:
			nl := len(v.lines)
			if nl > 0 {
				v.lines[nl-1] = append(v.lines[nl-1], ch)
			} else {
				v.lines = append(v.lines, []rune{ch})
			}
		}
	}
	return len(p), nil
}

// Read reads data into p. It returns the number of bytes read into p.
// At EOF, err will be io.EOF. Calling Read() after Rewind() makes the
// cache to be refreshed with the contents of the view.
func (v *View) Read(p []byte) (n int, err error) {
	if v.readOffset == 0 {
		v.readCache = v.Buffer()
	}
	if v.readOffset < len(v.readCache) {
		n = copy(p, v.readCache[v.readOffset:])
		v.readOffset += n
	} else {
		err = io.EOF
	}
	return
}

// Rewind sets the offset for the next Read to 0, which also refresh the
// read cache.
func (v *View) Rewind() {
	v.readOffset = 0
}

// draw re-draws the view's contents.
func (v *View) draw() error {
	maxX, maxY := v.Size()
	if v.Wrap {
		maxX--
	}

	if v.Wrap {
		if maxX == 0 {
			return errors.New("X size of the view cannot be 0")
		}
		v.ox = 0
	}
	if v.tainted {
		v.viewLines = nil
		for i, line := range v.lines {
			if v.Wrap {
				if len(line) <= maxX {
					vline := viewLine{linesX: 0, linesY: i, line: line}
					v.viewLines = append(v.viewLines, vline)
					continue
				} else {
					vline := viewLine{linesX: 0, linesY: i, line: line[:maxX]}
					v.viewLines = append(v.viewLines, vline)
				}
				// Append remaining lines
				for n := maxX; n < len(line); n += maxX {
					if len(line[n:]) <= maxX {
						vline := viewLine{linesX: n, linesY: i, line: line[n:]}
						v.viewLines = append(v.viewLines, vline)
					} else {
						vline := viewLine{linesX: n, linesY: i, line: line[n : n+maxX]}
						v.viewLines = append(v.viewLines, vline)
					}
				}
			} else {
				vline := viewLine{linesX: 0, linesY: i, line: line}
				v.viewLines = append(v.viewLines, vline)
			}
		}
		v.tainted = false
	}

	if v.Autoscroll && len(v.viewLines) > maxY {
		v.oy = len(v.viewLines) - maxY
	}
	y := 0
	for i, vline := range v.viewLines {
		if i < v.oy {
			continue
		}
		if y >= maxY {
			break
		}
		x := 0
		for j, ch := range vline.line {
			if j < v.ox {
				continue
			}
			if x >= maxX {
				break
			}
			if err := v.setRune(x, y, ch); err != nil {
				return err
			}
			x++
		}
		y++
	}
	return nil
}

// realPosition returns the position in the internal buffer corresponding to the
// point (x, y) of the view.
func (v *View) realPosition(vx, vy int) (x, y int, err error) {
	vx = v.ox + vx
	vy = v.oy + vy

	if vx < 0 || vy < 0 {
		return 0, 0, errors.New("invalid point")
	}

	if len(v.viewLines) == 0 {
		return vx, vy, nil
	}

	if vy < len(v.viewLines) {
		vline := v.viewLines[vy]
		x = vline.linesX + vx
		y = vline.linesY
	} else {
		vline := v.viewLines[len(v.viewLines)-1]
		x = vx
		y = vline.linesY + vy - len(v.viewLines) + 1
	}

	return x, y, nil
}

// Clear empties the view's internal buffer.
func (v *View) Clear() {
	v.tainted = true

	v.lines = nil
	v.clearRunes()
}

// clearRunes erases all the cells in the view.
func (v *View) clearRunes() {
	maxX, maxY := v.Size()
	for x := 0; x < maxX; x++ {
		for y := 0; y < maxY; y++ {
			termbox.SetCell(v.x0+x+1, v.y0+y+1, ' ',
				termbox.Attribute(v.FgColor), termbox.Attribute(v.BgColor))
		}
	}
}

func (v *View) absWriteRune(x, y int, ch rune) error {
	v.tainted = true

	if x < 0 || y < 0 {
		return errors.New("invalid point")
	}

	if y >= len(v.lines) {
		s := make([][]rune, y-len(v.lines)+1)
		v.lines = append(v.lines, s...)
	}

	olen := len(v.lines[y])
	if x >= len(v.lines[y]) {
		s := make([]rune, x-len(v.lines[y])+1)
		v.lines[y] = append(v.lines[y], s...)
	}
	if !v.Overwrite && x < olen {
		v.lines[y] = append(v.lines[y], '\x00')
		copy(v.lines[y][x+1:], v.lines[y][x:])
	}
	v.lines[y][x] = ch
	return nil
}

// writeRune writes a rune into the view's internal buffer, at the
// position corresponding to the point (x, y). The length of the internal
// buffer is increased if the point is out of bounds. Overwrite mode is
// governed by the value of View.overwrite.
func (v *View) writeRune(x, y int, ch rune) error {
	rx, ry, err := v.realPosition(x, y)
	if err != nil {
		return err
	}
	return v.absWriteRune(rx, ry, ch)
}

func (v *View) absDeleteRune(x, y int) error {
	v.tainted = true

	if x < 0 || y < 0 || y >= len(v.lines) || x >= len(v.lines[y]) {
		return errors.New("invalid point")
	}
	v.lines[y] = append(v.lines[y][:x], v.lines[y][x+1:]...)
	return nil
}

// deleteRune removes a rune from the view's internal buffer, at the
// position corresponding to the point (x, y).
func (v *View) deleteRune(x, y int) error {

	rx, ry, err := v.realPosition(x, y)
	if err != nil {
		return err
	}
	return v.absDeleteRune(rx, ry)
}

func (v *View) absMergeLines(y int) error {
	v.tainted = true

	if y < 0 || y >= len(v.lines) {
		return errors.New("invalid point")
	}

	if y < len(v.lines)-1 { // otherwise we don't need to merge anything
		v.lines[y] = append(v.lines[y], v.lines[y+1]...)
		v.lines = append(v.lines[:y+1], v.lines[y+2:]...)
		return nil
	}
	return errors.New("last line")
}

// mergeLines merges the lines "y" and "y+1" if possible.
func (v *View) mergeLines(y int) error {
	_, ry, err := v.realPosition(0, y)
	if err != nil {
		return err
	}
	return v.absMergeLines(ry)
}

func (v *View) absBreakLine(x, y int) error {
	v.tainted = true

	if y < 0 || y >= len(v.lines) {
		return errors.New("invalid point")
	}

	var left, right []rune
	if x < len(v.lines[y]) { // break line
		left = make([]rune, len(v.lines[y][:x]))
		copy(left, v.lines[y][:x])
		right = make([]rune, len(v.lines[y][x:]))
		copy(right, v.lines[y][x:])
	} else { // new empty line
		left = v.lines[y]
	}

	lines := make([][]rune, len(v.lines)+1)
	lines[y] = left
	lines[y+1] = right
	copy(lines, v.lines[:y])
	copy(lines[y+2:], v.lines[y+1:])
	v.lines = lines
	return nil
}

// breakLine breaks a line of the internal buffer at the position corresponding
// to the point (x, y).
func (v *View) breakLine(x, y int) error {
	rx, ry, err := v.realPosition(x, y)
	if err != nil {
		return err
	}
	return v.absBreakLine(rx, ry)
}

// Buffer returns a string with the contents of the view's internal
// buffer.
func (v *View) Buffer() string {
	str := ""
	for _, l := range v.lines {
		str += string(l) + "\n"
	}
	return strings.Replace(str, "\x00", " ", -1)
}

// ViewBuffer returns a string with the contents of the view's buffer that is
// shown to the user.
func (v *View) ViewBuffer() string {
	str := ""
	for _, l := range v.viewLines {
		str += string(l.line) + "\n"
	}
	return strings.Replace(str, "\x00", " ", -1)
}

// Line returns a string with the line of the view's internal buffer
// at the position corresponding to the point (x, y).
func (v *View) Line(y int) (string, error) {
	_, y, err := v.realPosition(0, y)
	if err != nil {
		return "", err
	}

	if y < 0 || y >= len(v.lines) {
		return "", errors.New("invalid point")
	}
	return string(v.lines[y]), nil
}

// Word returns a string with the word of the view's internal buffer
// at the position corresponding to the point (x, y).
func (v *View) Word(x, y int) (string, error) {
	x, y, err := v.realPosition(x, y)
	if err != nil {
		return "", err
	}

	if x < 0 || y < 0 || y >= len(v.lines) || x >= len(v.lines[y]) {
		return "", errors.New("invalid point")
	}
	l := string(v.lines[y])
	nl := strings.LastIndexFunc(l[:x], indexFunc)
	if nl == -1 {
		nl = 0
	} else {
		nl = nl + 1
	}
	nr := strings.IndexFunc(l[x:], indexFunc)
	if nr == -1 {
		nr = len(l)
	} else {
		nr = nr + x
	}
	return string(l[nl:nr]), nil
}

// indexFunc allows to split lines by words taking into account spaces
// and 0.
func indexFunc(r rune) bool {
	return r == ' ' || r == 0
}

func (v *View) SetEditable(b bool) {
	v.Editable = b
}
func (v *View) IsEditable() bool {
	return v.Editable
}

// permutLines permuts y1 and y2
func (v *View) permutLines(y1, y2 int) error {
	if y1 < 0 || y2 < 0 || y1 >= len(v.lines) || y2 >= len(v.lines) {
		return errors.New("invalid line")
	}

	s := v.lines[y1]
	v.lines[y1] = v.lines[y2]
	v.lines[y2] = s

	return nil
}
