// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocui

import (
	"errors"
	"fmt"

	"github.com/nsf/termbox-go"
)

// Handler represents a handler that can be used to update or modify the GUI.
type Handler func(*Gui) error

// userEvent represents an event triggered by the user.
type userEvent struct {
	h Handler
}

var (
	// ErrQuit is used to decide if the MainLoop finished successfully.
	ErrQuit = errors.New("quit")

	// ErrUnknownView allows to assert if a View must be initialized.
	ErrUnknownView = errors.New("unknown view")
	// ErrUnknownViewNode allows to assert if a View must be initialized.
	ErrUnknownViewNode = errors.New("unknown view node")

	// ErrUnknowMode checks map initialization
	ErrUnknowMode = errors.New("unknown mode")
)

// Gui represents the whole User Interface, including the views, layouts
// and keybindings.
type Gui struct {
	tbEvents    chan termbox.Event
	userEvents  chan userEvent
	viewTree    *Container
	currentView *View
	layout      Handler
	modes       []*Mode
	currentMode *Mode
	maxX, maxY  int

	// workingView represents the view related to a file to work on
	workingView *View

	// BgColor and FgColor allow to configure the background and foreground
	// colors of the GUI.
	BgColor, FgColor Attribute

	// SelBgColor and SelFgColor are used to configure the background and
	// foreground colors of the selected line, when it is highlighted.
	SelBgColor, SelFgColor Attribute

	// If Cursor is true then the cursor is enabled.
	Cursor bool

	// If Mouse is true then mouse events will be enabled.
	Mouse bool

	// Editor allows to define the editor that manages the edition mode,
	// including keybindings or cursor behaviour. DefaultEditor is used by
	// default.
	Editor Editor
}

// NewGui returns a new Gui object.
func NewGui() *Gui {
	return &Gui{}
}

// Init initializes the library. This function must be called before
// any other functions.
func (g *Gui) Init() error {
	if err := termbox.Init(); err != nil {
		return err
	}
	g.tbEvents = make(chan termbox.Event, 20)
	g.userEvents = make(chan userEvent, 20)
	g.maxX, g.maxY = termbox.Size()
	g.BgColor = ColorBlack
	g.FgColor = ColorWhite
	g.Editor = DefaultEditor

	g.currentView = nil
	tree := Container{name: ""}
	g.viewTree = &tree

	return nil
}

// SetCurrentMode switches to the Mode with the given name
func (g *Gui) SetCurrentMode(name string) error {
	for _, m := range g.modes {
		if m.name == name {
			g.currentMode = m
			return nil
		}
	}
	return ErrUnknowMode
}

// CurrentMode returns the current mode
func (g *Gui) CurrentMode() *Mode {
	return g.currentMode
}

// Mode returns a pointer to the Mode with the given name, or error
// ErrUnknownMode if a Mode with that name does not exist.
func (g *Gui) Mode(name string) (*Mode, error) {
	for _, m := range g.modes {
		if m.name == name {
			g.currentMode = m
			return m, nil
		}
	}
	return nil, ErrUnknowMode
}

// AddMode creates a new mode
// does nothing if there is already a mode for this name
func (g *Gui) AddMode(name string, openFunc modeHandler, closeFunc modeHandler) {
	if _, err := g.Mode(name); err == nil {
		return
	}
	g.modes = append(g.modes, CreateMode(name, openFunc, closeFunc))
}

// Close finalizes the library. It should be called after a successful
// initialization and when gocui is not needed anymore.
func (g *Gui) Close() {
	termbox.Close()
}

// Size returns the terminal's size.
func (g *Gui) Size() (x, y int) {
	return g.maxX, g.maxY
}

// SetRune writes a rune at the given point, relative to the top-left
// corner of the terminal. It checks if the position is valid and applies
// the gui's colors.
func (g *Gui) SetRune(x, y int, ch rune) error {
	if x < 0 || y < 0 || x >= g.maxX || y >= g.maxY {
		return errors.New("invalid point")
	}
	termbox.SetCell(x, y, ch, termbox.Attribute(g.FgColor), termbox.Attribute(g.BgColor))
	return nil
}

// Rune returns the rune contained in the cell at the given position.
// It checks if the position is valid.
func (g *Gui) Rune(x, y int) (rune, error) {
	if x < 0 || y < 0 || x >= g.maxX || y >= g.maxY {
		return ' ', errors.New("invalid point")
	}
	c := termbox.CellBuffer()[y*g.maxX+x]
	return c.Ch, nil
}

// SetView creates a new view with its top-left corner at (x0, y0)
// and the bottom-right one at (x1, y1). If a view with the same name
// already exists, its dimensions are updated; otherwise, the error
// ErrUnknownView is returned, which allows to assert if the View must
// be initialized. It checks if the position is valid.
func (g *Gui) SetView(name string, father string, x0, y0, x1, y1 int) (*View, error) {
	if x0 >= x1 || y0 >= y1 {
		return nil, fmt.Errorf("invalide dim: %d, %d %s", y1, x1, name)
	}
	if name == "" {
		return nil, errors.New("invalid name")
	}

	if v, err := g.View(name); err == nil {
		v.x0 = x0
		v.y0 = y0
		v.x1 = x1
		v.y1 = y1
		v.tainted = true
		return v, nil
	}

	v := newView(name, x0, y0, x1, y1)
	v.BgColor, v.FgColor = g.BgColor, g.FgColor
	v.SelBgColor, v.SelFgColor = g.SelBgColor, g.SelFgColor
	c, err := g.ViewNode(father)
	if c == nil && err != ErrUnknownViewNode {
		return nil, err
	}
	c.childrens = append(c.childrens, v)
	return v, ErrUnknownView
}

// SetViewNode creates a new view with its top-left corner at (x0, y0)
// and the bottom-right one at (x1, y1). If a viewNode with the same name
// already exists, its dimensions are updated; otherwise, the error
// ErrUnknownViewNode is returned. It checks if the position is valid.
func (g *Gui) SetViewNode(name string, father string, x0, y0, x1, y1 int) error {
	if x0 >= x1 || y0 >= y1 {
		return fmt.Errorf("invalide dim: %d, %d %s", y1, x1, name)
	}
	if name == "" {
		return errors.New("invalid name")
	}

	if vn, err := g.ViewNode(name); err == nil {
		vn.x0 = x0
		vn.y0 = y0
		vn.x1 = x1
		vn.y1 = y1
		return nil
	}

	c := newViewNode(name, x0, y0, x1, y1)
	cfather, err := g.ViewNode(father)
	if c != nil && err != nil {
		return err
	}
	cfather.childrens = append(cfather.childrens, c)
	return ErrUnknownViewNode
}

// SetViewOnTop sets the given view on top of the existing ones.
func (g *Gui) SetViewOnTop(name string) (*View, error) {
	if v, err := permuteGeom(g.viewTree, name); v != nil && err != ErrUnknownView {
		return v, err
	}
	return nil, ErrUnknownView
}

func permuteGeom(c *Container, name string) (*View, error) {
	for i, node := range c.childrens {
		if v, ok := node.(*View); ok && v.Name() == name {
			s := append(c.childrens[:i], c.childrens[i+1:]...)
			c.childrens = append(s, v)
			return v, nil
		} else if n, ok := node.(*Container); ok {
			v, err := permuteGeom(n, name)
			if v != nil && err != ErrUnknownView {
				s := append(c.childrens[:i], c.childrens[i+1:]...)
				c.childrens = append(s, n)
				return v, err
			}
		}
	}
	return nil, ErrUnknownView
}

// View returns a pointer to the view with the given name, or error
// ErrUnknownView if a view with that name does not exist.
func (g *Gui) View(name string) (*View, error) {
	return findView(g.viewTree, name)
}

func findView(c *Container, name string) (*View, error) {
	for _, node := range c.childrens {
		if v, ok := node.(*View); ok && v.Name() == name {
			return v, nil
		} else if cont, ok := node.(*Container); ok {
			v, err := findView(cont, name)
			if v != nil && err != ErrUnknownView {
				return v, err
			}
		}
	}
	return nil, ErrUnknownView
}

// ViewNode returns a pointer to the view with the given name, or error
// ErrUnknownView if a view with that name does not exist.
func (g *Gui) ViewNode(name string) (*Container, error) {
	return findViewNode(g.viewTree, name)
}
func findViewNode(c *Container, name string) (*Container, error) {
	if c.Name() == name {
		return c, nil
	}
	for _, node := range c.childrens {
		if cont, ok := node.(*Container); ok {
			result, err := findViewNode(cont, name)
			if result != nil && err != ErrUnknownViewNode {
				return result, err
			}
		}
	}
	return nil, ErrUnknownViewNode
}

func (v *View) findGeometry(c *Container, name string) (geom, error) {
	if c.Name() == name {
		return c, nil
	}
	for _, node := range c.childrens {
		if v, ok := node.(*View); ok && v.Name() == name {
			return v, nil
		} else if cont, ok := node.(*Container); ok {
			result, err := findViewNode(cont, name)
			if result != nil {
				return result, err
			}
		}
	}
	return nil, ErrUnknownView
}

// ViewByPosition returns a pointer to a view matching the given position, or
// error ErrUnknownView if a view in that position does not exist.
func (g *Gui) ViewByPosition(x, y int) (*View, error) {
	if v, err := findViewByPosition(g.viewTree, x, y); v != nil && err == nil {
		return v, err
	}
	return nil, ErrUnknownView
}
func findViewByPosition(c *Container, x, y int) (*View, error) {
	for _, node := range c.childrens {
		if v, ok := node.(*View); ok && x > v.x0 && x < v.x1 && y > v.y0 && y < v.y1 {
			return v, nil
		} else if cont, ok := node.(*Container); ok {
			result, err := findViewByPosition(cont, x, y)
			if result != nil && err != ErrUnknownView {
				return result, err
			}
		}
	}
	return nil, ErrUnknownView
}

// ViewPosition returns the coordinates of the view with the given name, or
// error ErrUnknownView if a view with that name does not exist.
func (g *Gui) ViewPosition(name string) (x0, y0, x1, y1 int, err error) {
	if x0, y0, x1, y1, err := findViewPosition(g.viewTree, name); err == nil {
		return x0, y0, x1, y1, err
	}
	return 0, 0, 0, 0, ErrUnknownView
}
func findViewPosition(c *Container, name string) (x0, y0, x1, y1 int, err error) {
	for _, node := range c.childrens {
		if v, ok := node.(*View); ok && node.Name() == name {
			return v.x0, v.y0, v.x1, v.y1, nil
		} else if cont, ok := node.(*Container); ok {
			x0, y0, x1, y1, err := findViewPosition(cont, name)
			if err == nil {
				return x0, y0, x1, y1, err
			}
		}
	}
	return 0, 0, 0, 0, ErrUnknownView
}

// DeleteView deletes a view by name.
func (g *Gui) DeleteView(name string) error {
	return deleteViewRecursive(g.viewTree, name)
}

func deleteViewRecursive(c *Container, name string) error {
	for i, node := range c.childrens {
		if _, ok := node.(*View); ok && node.Name() == name {
			c.childrens = append(c.childrens[:i], c.childrens[i+1:]...)
			return nil
		} else if cont, ok := node.(*Container); ok {
			err := deleteViewRecursive(cont, name)
			if err == nil {
				return nil
			}
		}
	}
	return ErrUnknownView
}

// SetCurrentView gives the focus to a given view.
func (g *Gui) SetCurrentView(name string) error {
	if v, err := g.View(name); err == nil {
		g.currentView = v
		return nil
	}
	return ErrUnknownView
}

// CurrentView returns the currently focused view, or nil if no view
// owns the focus.
func (g *Gui) CurrentView() *View {
	return g.currentView
}

// Workingview returns the currently working view, or nil if no view
// owns the focus.
func (g *Gui) Workingview() *View {
	return g.workingView
}

// SetWorkingView gives the focus to a given view.
func (g *Gui) SetWorkingView(name string) error {
	if v, err := g.View(name); err == nil {
		g.workingView = v
		return nil
	}
	return ErrUnknownView
}

// SetKeybinding creates a new keybinding. If viewname equals to ""
// (empty string) then the keybinding will apply to all views. key must
// be a rune or a Key.
func (g *Gui) SetKeybinding(modeName string, viewName string, key interface{}, mod Modifier, h KeybindingHandler) error {
	var kb *keybinding

	switch k := key.(type) {
	case Key:
		kb = newKeybinding(viewName, k, 0, mod, h)
	case rune:
		kb = newKeybinding(viewName, 0, k, mod, h)
	default:
		return errors.New("unknown type")
	}

	if m, err := g.Mode(modeName); err == nil {
		*m.GetKeyBindings() = append(*m.GetKeyBindings(), kb)
	}
	return nil
}

// Execute executes the given handler. This function can be called safely from
// a goroutine in order to update the GUI. It is important to note that it
// won't be executed immediately, instead it will be added to the user events
// queue.
func (g *Gui) Execute(h Handler) {
	go func() { g.userEvents <- userEvent{h: h} }()
}

// SetLayout sets the current layout. A layout is a function that
// will be called every time the gui is redrawn, it must contain
// the base views and its initializations.
func (g *Gui) SetLayout(layout Handler) {
	g.layout = layout
	go func() { g.tbEvents <- termbox.Event{Type: termbox.EventResize} }()
}

// MainLoop runs the main loop until an error is returned. A successful
// finish should return ErrQuit.
func (g *Gui) MainLoop() error {
	go func() {
		for {
			g.tbEvents <- termbox.PollEvent()
		}
	}()

	inputMode := termbox.InputEsc
	if g.Mouse {
		inputMode |= termbox.InputMouse
	}
	termbox.SetInputMode(inputMode)

	if err := g.flush(); err != nil {
		return err
	}
	for {

		select {
		case ev := <-g.tbEvents:
			if err := g.handleEvent(&ev); err != nil {
				return err
			}
		case ev := <-g.userEvents:
			if err := ev.h(g); err != nil {
				return err
			}
		}

		if err := g.consumeevents(); err != nil {
			return err
		}
		if err := g.flush(); err != nil {
			return err
		}
	}
}

// consumeevents handles the remaining events in the events pool.
func (g *Gui) consumeevents() error {
	for {
		select {
		case ev := <-g.tbEvents:
			if err := g.handleEvent(&ev); err != nil {
				return err
			}
		case ev := <-g.userEvents:
			if err := ev.h(g); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

// handleEvent handles an event, based on its type (key-press, error,
// etc.)
func (g *Gui) handleEvent(ev *termbox.Event) error {
	switch ev.Type {
	case termbox.EventKey, termbox.EventMouse:
		return g.onKey(ev)
	case termbox.EventError:
		return ev.Err
	default:
		return nil
	}
}

// flush updates the gui, re-drawing frames and buffers.
func (g *Gui) flush() error {
	if g.layout == nil {
		return errors.New("Null layout")
	}

	termbox.Clear(termbox.Attribute(g.FgColor), termbox.Attribute(g.BgColor))

	maxX, maxY := termbox.Size()
	// if GUI's size has changed, we need to redraw all views
	if maxX != g.maxX || maxY != g.maxY {
		updateViews(g.viewTree)
	}
	g.maxX, g.maxY = maxX, maxY

	if err := g.layout(g); err != nil {
		return err
	}
	g.displayViews(g.viewTree)

	if err := g.drawIntersections(); err != nil {
		return err
	}
	termbox.Flush()
	return nil
}

func updateViews(c *Container) {
	for _, node := range c.childrens {
		if v, ok := node.(*View); ok {
			v.tainted = true
		} else if cont, ok := node.(*Container); ok {
			updateViews(cont)
		}
	}
}

func (g *Gui) displayViews(c *Container) error {
	for _, node := range c.childrens {
		if v, ok := node.(*View); ok {
			if v.Frame {
				if err := g.drawFrame(v); err != nil {
					return err
				}
				if v.Title != "" {
					if err := g.drawTitle(v); err != nil {
						return err
					}
				}
				if v.Footer != "" {
					if err := g.drawFooter(v); err != nil {
						return err
					}
				}
			}

			if err := g.draw(v); err != nil {
				return err
			}
		} else if cont, ok := node.(*Container); ok {
			if err := g.displayViews(cont); err != nil {
				return err
			}
		}
	}
	return nil
}

// drawFrame draws the horizontal and vertical edges of a view.
func (g *Gui) drawFrame(v *View) error {
	for x := v.x0 + 1; x < v.x1 && x < g.maxX; x++ {
		if x < 0 {
			continue
		}
		if v.y0 > -1 && v.y0 < g.maxY {
			if err := g.SetRune(x, v.y0, '─'); err != nil {
				return err
			}
		}
		if v.y1 > -1 && v.y1 < g.maxY {
			if err := g.SetRune(x, v.y1, '─'); err != nil {
				return err
			}
		}
	}
	for y := v.y0 + 1; y < v.y1 && y < g.maxY; y++ {
		if y < 0 {
			continue
		}
		if v.x0 > -1 && v.x0 < g.maxX {
			if err := g.SetRune(v.x0, y, '│'); err != nil {
				return err
			}
		}
		if v.x1 > -1 && v.x1 < g.maxX {
			if err := g.SetRune(v.x1, y, '│'); err != nil {
				return err
			}
		}
	}
	return nil
}

// drawTitle draws the title of the view.
func (g *Gui) drawTitle(v *View) error {
	if v.y0 < 0 || v.y0 >= g.maxY {
		return nil
	}

	for i, ch := range v.Title {
		x := v.x0 + i + 2
		if x < 0 {
			continue
		} else if x > v.x1-2 || x >= g.maxX {
			break
		}
		if err := g.SetRune(x, v.y0, ch); err != nil {
			return err
		}
	}
	return nil
}

// drawFooter draws the footer of the view.
func (g *Gui) drawFooter(v *View) error {
	if v.y1 < 0 || v.y1 >= g.maxY {
		return nil
	}

	for i, ch := range v.Footer {
		x := v.x1 + i - 2 - len(v.Footer)
		if x < 0 {
			continue
		} else if x > v.x1-2 || x >= g.maxX {
			break
		}
		if err := g.SetRune(x, v.y1, ch); err != nil {
			return err
		}
	}
	return nil
}

// draw manages the cursor and calls the draw function of a view.
func (g *Gui) draw(v geom) error {
	if g.Cursor {
		if v := g.currentView; v != nil {
			vMaxX, vMaxY := v.Size()
			if v.cx < 0 {
				v.cx = 0
			} else if v.cx >= vMaxX {
				v.cx = vMaxX - 1
			}
			if v.cy < 0 {
				v.cy = 0
			} else if v.cy >= vMaxY {
				v.cy = vMaxY - 1
			}

			gMaxX, gMaxY := g.Size()
			cx, cy := v.x0+v.cx+1, v.y0+v.cy+1
			if cx >= 0 && cx < gMaxX && cy >= 0 && cy < gMaxY {
				termbox.SetCursor(cx, cy)
			} else {
				termbox.HideCursor()
			}
		}
	} else {
		termbox.HideCursor()
	}

	if a, ok := v.(*View); ok {
		a.clearRunes()
	}
	if err := v.draw(); err != nil {
		return err
	}
	return nil
}

// drawIntersections draws the corners of each view, based on the type
// of the edges that converge at these points.
func (g *Gui) drawIntersections() error {
	return g.drawIntersectionsRecursively(g.viewTree)
}

func (g *Gui) drawIntersectionsRecursively(c *Container) error {
	for _, node := range c.childrens {
		if v, ok := node.(*View); ok {
			if ch, ok := g.intersectionRune(v.x0, v.y0); ok {
				if err := g.SetRune(v.x0, v.y0, ch); err != nil {
					return err
				}
			}
			if ch, ok := g.intersectionRune(v.x0, v.y1); ok {
				if err := g.SetRune(v.x0, v.y1, ch); err != nil {
					return err
				}
			}
			if ch, ok := g.intersectionRune(v.x1, v.y0); ok {
				if err := g.SetRune(v.x1, v.y0, ch); err != nil {
					return err
				}
			}
			if ch, ok := g.intersectionRune(v.x1, v.y1); ok {
				if err := g.SetRune(v.x1, v.y1, ch); err != nil {
					return err
				}
			}
		} else if cont, ok := node.(*Container); ok {
			err := g.drawIntersectionsRecursively(cont)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// intersectionRune returns the correct intersection rune at a given
// point.
func (g *Gui) intersectionRune(x, y int) (rune, bool) {
	if x < 0 || y < 0 || x >= g.maxX || y >= g.maxY {
		return ' ', false
	}

	chTop, _ := g.Rune(x, y-1)
	top := verticalRune(chTop)
	chBottom, _ := g.Rune(x, y+1)
	bottom := verticalRune(chBottom)
	chLeft, _ := g.Rune(x-1, y)
	left := horizontalRune(chLeft)
	chRight, _ := g.Rune(x+1, y)
	right := horizontalRune(chRight)

	var ch rune
	switch {
	case !top && bottom && !left && right:
		ch = '┌'
	case !top && bottom && left && !right:
		ch = '┐'
	case top && !bottom && !left && right:
		ch = '└'
	case top && !bottom && left && !right:
		ch = '┘'
	case top && bottom && left && right:
		ch = '┼'
	case top && bottom && !left && right:
		ch = '├'
	case top && bottom && left && !right:
		ch = '┤'
	case !top && bottom && left && right:
		ch = '┬'
	case top && !bottom && left && right:
		ch = '┴'
	default:
		return ' ', false
	}
	return ch, true
}

// verticalRune returns if the given character is a vertical rune.
func verticalRune(ch rune) bool {
	if ch == '│' || ch == '┼' || ch == '├' || ch == '┤' {
		return true
	}
	return false
}

// verticalRune returns if the given character is a horizontal rune.
func horizontalRune(ch rune) bool {
	if ch == '─' || ch == '┼' || ch == '┬' || ch == '┴' {
		return true
	}
	return false
}

// onKey manages key-press events. A keybinding handler is called when
// a key-press or mouse event satisfies a configured keybinding. Furthermore,
// currentView's internal buffer is modified if currentView.Editable is true.
func (g *Gui) onKey(ev *termbox.Event) error {
	var curView *View

	switch ev.Type {
	case termbox.EventKey:
		if g.currentView != nil && g.currentView.Editable && g.Editor != nil {
			g.Editor.Edit(g.currentView, Key(ev.Key), ev.Ch, Modifier(ev.Mod))
		}
		curView = g.currentView
	case termbox.EventMouse:
		mx, my := ev.MouseX, ev.MouseY
		v, err := g.ViewByPosition(mx, my)
		if err != nil {
			break
		}
		if err := v.SetCursor(mx-v.x0-1, my-v.y0-1); err != nil {
			return err
		}
		curView = v
	}

	for _, kb := range g.currentMode.keybindings {
		if kb.h == nil {
			continue
		}
		if kb.matchKeypress(Key(ev.Key), ev.Ch, Modifier(ev.Mod)) && kb.matchView(g.viewTree, curView) {
			if err := kb.h(g, curView); err != nil {
				return err
			}
		}
	}

	g.UpdateHistoric()
	return nil
}
