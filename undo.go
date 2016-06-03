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
	undoSt CmdStack
	redoSt CmdStack
}

type WriteCmd struct {
	v    *View
	x, y int // position in the lines (not the viewlines)
	p    []rune
}

type BackDeleteCmd struct {
	v    *View
	x, y int
	p    []rune // deleted
}

type NewLineCmd struct {
	v    *View
	x, y int
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

func (con *Context) Exec(c Command) {

	if _, ok := c.(Mergeable); ok {
		if l := len(con.undoSt); l > 0 {
			pr := con.undoSt[l-1]
			if reflect.TypeOf(pr) == reflect.TypeOf(c) {
				pr.(Mergeable).merge(c.(Mergeable))
				return
			}
		}
	}
	con.undoSt.Push(c)
	con.redoSt.Clear()
}

func (con *Context) Undo() {
	if c := con.undoSt.Pop(); c != nil {
		con.redoSt.Push(c)
		c.Reverse()
	}
}

func (con *Context) Redo() {
	if c := con.redoSt.Pop(); c != nil {
		con.undoSt.Push(c)
		c.Execute()
	}
}

func NewWriteCmd(v *View, x, y int, fchar rune) *WriteCmd {
	return &WriteCmd{v: v, x: x, y: y, p: []rune{fchar}}
}

func NewBackDeleteCmd(v *View, x, y int, fchar rune) *BackDeleteCmd {
	return &BackDeleteCmd{v: v, x: x, y: y, p: []rune{fchar}}
}

func NewNewLineCmd(v *View, x, y int) *NewLineCmd {
	return &NewLineCmd{v: v, x: x, y: y}
}

func NewSpaceCmd(v *View, x, y int) *SpaceCmd {
	return &SpaceCmd{v: v, x: x, y: y, n: 1}
}

func (c *BackDeleteCmd) Execute() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	for i := 0; i < len(c.p); i++ {
		c.v.EditDelete(true)
	}
}

func (c *BackDeleteCmd) Reverse() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	for _, ch := range c.p {
		c.v.writeRune(c.v.cx, c.v.cy, ch)
	}
}

func (c *BackDeleteCmd) merge(m Mergeable) {
	if o, ok := m.(*BackDeleteCmd); ok {
		c.p = append(o.p, c.p...)
	}
}

func (c *NewLineCmd) Execute() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	c.v.breakLine(c.v.cx, c.v.cy)
}

func (c *NewLineCmd) Reverse() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	c.v.mergeLines(c.y)
}

func (c *SpaceCmd) Info() string {
	return fmt.Sprintf("Space : %d times", c.n)
}
func (c *NewLineCmd) Info() string {
	return "NewLine"
}

func (c *BackDeleteCmd) Info() string {
	return "Delete : " + string(c.p)
}

type SpaceCmd struct {
	v    *View
	x, y int
	n    int // number of space
}

func (c *SpaceCmd) Execute() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	for i := 0; i < c.n; i++ {
		c.v.writeRune(c.v.cx, c.v.cy, ' ')

	}
	c.v.MoveCursor(c.n, 0, false)
}

func (c *SpaceCmd) Reverse() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	for i := 0; i < c.n; i++ {
		c.v.EditDelete(false)
	}
}

func (c *SpaceCmd) merge(m Mergeable) {
	if _, ok := m.(*SpaceCmd); ok {
		c.n++
	}
}

func (c *WriteCmd) merge(m Mergeable) {
	if o, ok := m.(*WriteCmd); ok {
		c.p = append(c.p, o.p...)
	}
}

func (c *WriteCmd) Execute() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	for _, ch := range c.p {
		c.v.writeRune(c.v.cx, c.v.cy, ch)
	}
	c.v.MoveCursor(len(c.p), 0, false)
}

func (c *WriteCmd) Reverse() {
	c.v.SetCursor(0, 0)
	moveTo(c.v, c.x, c.y)
	for i := 0; i < len(c.p); i++ {
		c.v.EditDelete(false)
	}
}

func (c *WriteCmd) Info() string {
	return "Write : " + string(c.p)
}

func (con *Context) ToString() string {
	s := make([]byte, 1000) // todo : change the length
	l := 0

	l += copy(s[l:], "---UNDO---\n")

	for _, c := range con.undoSt {
		l += copy(s[l:], c.Info()+"\n")
	}

	l += copy(s[l:], "---REDO---\n")

	for i := len(con.redoSt) - 1; i >= 0; i-- {
		l += copy(s[l:], con.redoSt[i].Info()+"\n")
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

	vh.Clear()

	fmt.Fprint(vh, vm.Actions.ToString())
}
