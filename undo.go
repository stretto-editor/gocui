package gocui

import (
	"fmt"
	"reflect"
)

type Command interface {
	Info() string
	Execute()
	Reverse()
}

type Mergeable interface {
	merge(m Mergeable)
}

type ActionsInterface interface {
	Exec(c Command)
	Undo()
	Redo()
}

type CmdStack []Command

type Context struct {
	merge  bool
	undoSt CmdStack
	redoSt CmdStack
}

func (s *CmdStack) Push(c Command) {
	*s = append(*s, c)
}

func (s *CmdStack) Pop() Command {
	le := len(*s)
	if le < 1 {
		return nil
	}
	ret := (*s)[le-1]
	*s = (*s)[0 : le-1]
	return ret
}

func (s *CmdStack) Clear() {
	*s = nil
}

func (con *Context) Cut() {
	con.merge = false
}

func (con *Context) Exec(c Command) {
	if con.merge {
		if _, ok := c.(Mergeable); ok {
			if l := len(con.undoSt); l > 0 {
				pr := con.undoSt[l-1]
				if reflect.TypeOf(pr) == reflect.TypeOf(c) {
					pr.(Mergeable).merge(c.(Mergeable))
					return
				}
			}
		}
	}
	con.merge = true
	con.undoSt.Push(c)
	con.redoSt.Clear()
}

func (con *Context) Undo() {
	if c := con.undoSt.Pop(); c != nil {
		con.redoSt.Push(c)
		con.merge = false
		c.Reverse()
	}
}

func (con *Context) Redo() {
	if c := con.redoSt.Pop(); c != nil {
		con.undoSt.Push(c)
		con.merge = false
		c.Execute()
	}
}

// ---------------------- PASTE CMD ------------------------- //
/*
type PasteCmd struct {
	v    *View
	x, y int
	act  []Command
}

func NewPasteCmd(v *View, x, y int) *PasteCmd {
	return &PasteCmd{v: v, x: x, y: y}
}

func (c *PasteCmd) Execute() {
	for _, cmd := range c.act {
		cmd.Execute()
	}
}

func (c *PasteCmd) Reverse() {
	for i, _ := range c.act {
		c.act[i].Reverse()
	}
}

func (c *PasteCmd) Info() string {
	return "Paste"
}

func (c *PasteCmd) merge(m Mergeable) {
	if o, ok := m.(*Command); ok {
		c.act = append(c.act, o)
	}
}
*/
// ---------------------- DELLINE CMD ------------------------- //

type DelLineCmd struct {
	v    *View
	x, y int
	n    int
}

func NewDelLineCmd(v *View, x, y int) *DelLineCmd {
	return &DelLineCmd{v: v, x: x, y: y, n: 1}
}

func (c *DelLineCmd) Execute() {
	for i := 0; i < c.n; i++ {
		c.v.properMergeLines(c.y - 1)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x-c.n, c.y, false)
}

func (c *DelLineCmd) Reverse() {
	for i := 0; i < c.n; i++ {
		c.v.properBreakLine(0, c.y-c.n+1)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(0, c.y, false)
}

func (c *DelLineCmd) Info() string {
	return fmt.Sprintf("%d DelLine(s)", c.n)
}

func (c *DelLineCmd) merge(m Mergeable) {
	if _, ok := m.(*DelLineCmd); ok {
		c.n++
	}
}

// ---------------------- BACKDEL CMD ------------------------- //

type BackDeleteCmd struct {
	v    *View
	x, y int
	p    []rune // deleted
}

func NewBackDeleteCmd(v *View, x, y int, fchar rune) *BackDeleteCmd {
	return &BackDeleteCmd{v: v, x: x, y: y, p: []rune{fchar}}
}

func (c *BackDeleteCmd) Execute() {
	for i := 0; i < len(c.p); i++ {
		c.v.properDeleteRune(c.x-i-1, c.y)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x-len(c.p)+1, c.y, false)
}

func (c *BackDeleteCmd) Reverse() {
	for i, ch := range c.p {
		c.v.properWriteRune(c.x-len(c.p)+i, c.y, ch)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x+1, c.y, false)
}

func (c *BackDeleteCmd) Info() string {
	return "Delete : " + string(c.p)
}

func (c *BackDeleteCmd) merge(m Mergeable) {
	if o, ok := m.(*BackDeleteCmd); ok {
		c.p = append(o.p, c.p...)
	}
}

// ---------------------- FWDDEL CMD ------------------------- //

type FwdDeleteCmd struct {
	v    *View
	x, y int
	p    []rune // deleted
}

func NewFwdDeleteCmd(v *View, x, y int, fchar rune) *FwdDeleteCmd {
	return &FwdDeleteCmd{v: v, x: x, y: y, p: []rune{fchar}}
}

func (c *FwdDeleteCmd) Execute() {
	for i := 0; i < len(c.p); i++ {
		c.v.properDeleteRune(c.x, c.y)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x-len(c.p)+1, c.y, false)
}

func (c *FwdDeleteCmd) Reverse() {
	for i := len(c.p) - 1; i >= 0; i-- {
		c.v.properWriteRune(c.x, c.y, c.p[i])
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x+len(c.p), c.y, false)
}

func (c *FwdDeleteCmd) Info() string {
	return "Delete : " + string(c.p)
}

func (c *FwdDeleteCmd) merge(m Mergeable) {
	if o, ok := m.(*FwdDeleteCmd); ok {
		c.p = append(c.p, o.p...)
	}
}

// ---------------------- NEWLINE CMD ------------------------- //

type NewLineCmd struct {
	v    *View
	x, y int
	n    int // number of new lines
}

func NewNewLineCmd(v *View, x, y int) *NewLineCmd {
	return &NewLineCmd{v: v, x: x, y: y, n: 1}
}

func (c *NewLineCmd) Execute() {
	for i := 0; i < c.n; i++ {
		c.v.properBreakLine(c.x, c.y)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x+c.n, c.y, false) // should work with c.v.MoveCursor(c.x+c.n, c.y, false)
}

func (c *NewLineCmd) Reverse() {
	for i := 0; i < c.n; i++ {
		c.v.properMergeLines(c.y)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x, c.y, false)
}

func (c *NewLineCmd) Info() string {
	return fmt.Sprintf("%d NewLine(s)", c.n)
}

func (c *NewLineCmd) merge(m Mergeable) {
	if _, ok := m.(*NewLineCmd); ok {
		c.n++
	}
}

// ---------------------- SPACE CMD ------------------------- //

type SpaceCmd struct {
	v    *View
	x, y int
	n    int // number of space
}

func NewSpaceCmd(v *View, x, y int) *SpaceCmd {
	return &SpaceCmd{v: v, x: x, y: y, n: 1}
}

func (c *SpaceCmd) Execute() {
	for i := 0; i < c.n; i++ {
		c.v.properWriteRune(c.x, c.y, ' ')
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x+c.n+1, c.y, false)
}

func (c *SpaceCmd) Reverse() {
	for i := 0; i < c.n; i++ {
		c.v.properDeleteRune(c.x, c.y)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x+1, c.y, false)
}

func (c *SpaceCmd) Info() string {
	return fmt.Sprintf("%d Spaces", c.n)
}

func (c *SpaceCmd) merge(m Mergeable) {
	if _, ok := m.(*SpaceCmd); ok {
		c.n++
	}
}

// ---------------------- WRITE CMD ------------------------- //

type WriteCmd struct {
	v    *View
	x, y int // position in the lines (not the viewlines)
	p    []rune
}

func NewWriteCmd(v *View, x, y int, fchar rune) *WriteCmd {
	return &WriteCmd{v: v, x: x, y: y, p: []rune{fchar}}
}

func (c *WriteCmd) Execute() {
	for i := len(c.p) - 1; i >= 0; i-- {
		c.v.properWriteRune(c.x, c.y, c.p[i])
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x+len(c.p)+1, c.y, false)
}

func (c *WriteCmd) Reverse() {
	for i := 0; i < len(c.p); i++ {
		c.v.properDeleteRune(c.x, c.y)
	}
	c.v.SetOrigin(0, 0)
	c.v.SetCursor(0, 0)
	c.v.MoveCursor(c.x+1, c.y, false)
}

func (c *WriteCmd) Info() string {
	return "Write : " + string(c.p)
}

func (c *WriteCmd) merge(m Mergeable) {
	if o, ok := m.(*WriteCmd); ok {
		c.p = append(c.p, o.p...)
	}
}

func (con *Context) ToString(w int) string {
	s := make([]byte, 2000) // todo : change the length
	l := 0

	l += copy(s[l:], "---UNDO---\n")

	for _, c := range con.undoSt {
		info := c.Info()
		if len(info) < w {
			l += copy(s[l:], info+"\n")
		} else {
			l += copy(s[l:], info[:w-3]+"..."+"\n")
		}
	}

	l += copy(s[l:], "---REDO---\n")

	for i := len(con.redoSt) - 1; i >= 0; i-- {
		info := con.redoSt[i].Info()
		if len(info) < w {
			l += copy(s[l:], info+"\n")
		} else {
			l += copy(s[l:], info[:w-3]+"..."+"\n")
		}
	}

	return string(s)
}

//
// TODO : following functions must and will be moved/deleted
//

func moveTo(v *View, x int, y int) error {
	_, yOrigin := v.Origin()
	_, ySize := v.Size()

	if y <= ySize {

		v.SetCursor(x, y)
		return nil
	}
	// how many times we move from the size of the window
	var i int
	for i = 0; y > ySize; i++ {
		y -= ySize

	}
	v.SetOrigin(0, yOrigin+i*ySize)
	v.SetCursor(x, y)
	return nil
}

func (g *Gui) UpdateHistoric() {
	// TODO : this should be in stretto
	var vm, vh *View

	vm, _ = g.View("main")
	vh, _ = g.View("historic")
	w, _ := vh.Size()

	vh.Clear()

	fmt.Fprint(vh, vm.Actions.ToString(w))
}
