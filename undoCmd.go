package gocui

import (
	"fmt"
)

/*
v is the view on which the command has been performed.

The x,y coordinates are the buffer-coordinates
of the cursor BEFORE commands execution.
In concrete terms, it means that x,y are the return values of
realPosition called on the cursor position before the actual edition.
*/

/*
// ---------------------- PASTE CMD ------------------------- //

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

// ---------------------- UPPERMUT CMD ------------------------- //

type UpPermutCmd struct {
	v    *View
	x, y int
	n    int // relative position of the destination line
}

func NewUpPermutCmd(v *View, x, y int, n int) *UpPermutCmd {
	return &UpPermutCmd{v: v, x: x, y: y, n: n}
}

func (c *UpPermutCmd) Execute() {
	for i := c.y; i > c.y+c.n; i-- {
		c.v.permutLines(i, i-1)
	}
	c.v.AbsMoveCursor(c.x, c.y+c.n, false)
}

func (c *UpPermutCmd) Reverse() {
	for i := c.y + c.n; i < c.y; i++ {
		c.v.permutLines(i, i+1)
	}
	c.v.AbsMoveCursor(c.x, c.y, false)
}

func (c *UpPermutCmd) Info() string {
	return fmt.Sprintf("MoveLine %d -> %d", c.y+1, c.y+c.n+1)
}

func (c *UpPermutCmd) merge(m Mergeable) {
	if o, ok := m.(*UpPermutCmd); ok {
		c.n += o.n
	}
}

// ---------------------- DOWNPERMUT CMD ------------------------- //

type DownPermutCmd struct {
	v    *View
	x, y int
	n    int // relative position of the destination line
}

func NewDownPermutCmd(v *View, x, y int, n int) *DownPermutCmd {
	return &DownPermutCmd{v: v, x: x, y: y, n: n}
}

func (c *DownPermutCmd) Execute() {
	for i := c.y; i < c.y+c.n; i++ {
		c.v.permutLines(i, i+1)
	}
	c.v.AbsMoveCursor(c.x, c.y+c.n, false)
}

func (c *DownPermutCmd) Reverse() {
	for i := c.y + c.n; i > c.y; i-- {
		c.v.permutLines(i, i-1)
	}
	c.v.AbsMoveCursor(c.x, c.y, false)
}

func (c *DownPermutCmd) Info() string {
	return fmt.Sprintf("MoveLine %d -> %d", c.y+1, c.y+c.n+1)
}

func (c *DownPermutCmd) merge(m Mergeable) {
	if o, ok := m.(*DownPermutCmd); ok {
		c.n += o.n
	}
}

// ---------------------- FWDDELLINE CMD ------------------------- //

type FwdDelLineCmd struct {
	v    *View
	x, y int
	n    int
}

func NewFwdDelLineCmd(v *View, x, y int) *FwdDelLineCmd {
	return &FwdDelLineCmd{v: v, x: x, y: y, n: 1}
}

func (c *FwdDelLineCmd) Execute() {
	for i := 0; i < c.n; i++ {
		c.v.absMergeLines(c.y)
	}
	c.v.AbsMoveCursor(c.x, c.y, false)
}

func (c *FwdDelLineCmd) Reverse() {
	for i := 0; i < c.n; i++ {
		c.v.absBreakLine(c.x, c.y)
	}
	c.v.AbsMoveCursor(c.x, c.y, false)
}

func (c *FwdDelLineCmd) Info() string {
	return fmt.Sprintf("%d FwdDelLine(s)", c.n)
}

func (c *FwdDelLineCmd) merge(m Mergeable) {
	if _, ok := m.(*FwdDelLineCmd); ok {
		c.n++
	}
}

// ---------------------- BACKDELLINE CMD ------------------------- //

type BackDelLineCmd struct {
	v *View
	// this is
	px, py int
	n      int
}

func NewBackDelLineCmd(v *View, px, py int) *BackDelLineCmd {
	return &BackDelLineCmd{v: v, px: px, py: py, n: 1}
}

func (c *BackDelLineCmd) Execute() {
	for i := 0; i < c.n; i++ {
		c.v.absMergeLines(c.py)
	}
	c.v.AbsMoveCursor(c.px, c.py, false)
}

func (c *BackDelLineCmd) Reverse() {
	c.v.absBreakLine(c.px, c.py)
	for i := 1; i < c.n; i++ {
		c.v.absBreakLine(0, c.py+i)
	}
	c.v.AbsMoveCursor(0, c.py+c.n, false)
}

func (c *BackDelLineCmd) Info() string {
	return fmt.Sprintf("%d DelLine(s)", c.n)
}

func (c *BackDelLineCmd) merge(m Mergeable) {
	if o, ok := m.(*BackDelLineCmd); ok {
		c.py = o.py
		c.px = o.px
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
		c.v.absDeleteRune(c.x-i-1, c.y)
	}
	c.v.AbsMoveCursor(c.x-len(c.p), c.y, false)
}

func (c *BackDeleteCmd) Reverse() {
	for i, ch := range c.p {
		c.v.absWriteRune(c.x-len(c.p)+i, c.y, ch)
	}
	c.v.AbsMoveCursor(c.x, c.y, false)
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
	p    []rune // word thats been deleted
}

func NewFwdDeleteCmd(v *View, x, y int, fchar rune) *FwdDeleteCmd {
	return &FwdDeleteCmd{v: v, x: x, y: y, p: []rune{fchar}}
}

func (c *FwdDeleteCmd) Execute() {
	for i := 0; i < len(c.p); i++ {
		c.v.absDeleteRune(c.x, c.y)
	}
	c.v.AbsMoveCursor(c.x-len(c.p)+1, c.y, false)
}

func (c *FwdDeleteCmd) Reverse() {
	for i := len(c.p) - 1; i >= 0; i-- {
		c.v.absWriteRune(c.x, c.y, c.p[i])
	}
	c.v.AbsMoveCursor(c.x+len(c.p), c.y, false)
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
	n    int // nb of successive newlines made
}

func NewNewLineCmd(v *View, x, y int) *NewLineCmd {
	return &NewLineCmd{v: v, x: x, y: y, n: 1}
}

func (c *NewLineCmd) Execute() {
	for i := 0; i < c.n; i++ {
		c.v.absBreakLine(c.x, c.y)
	}
	c.v.AbsMoveCursor(c.x+c.n, c.y, false)
}

func (c *NewLineCmd) Reverse() {
	for i := 0; i < c.n; i++ {
		c.v.absMergeLines(c.y)
	}
	c.v.AbsMoveCursor(c.x, c.y, false)
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
	n    int // nb of successive spaces made
}

func NewSpaceCmd(v *View, x, y int) *SpaceCmd {
	return &SpaceCmd{v: v, x: x, y: y, n: 1}
}

func (c *SpaceCmd) Execute() {
	for i := 0; i < c.n; i++ {
		c.v.absWriteRune(c.x, c.y, ' ')
	}
	c.v.AbsMoveCursor(c.x+c.n+1, c.y, false)
}

func (c *SpaceCmd) Reverse() {
	for i := 0; i < c.n; i++ {
		c.v.absDeleteRune(c.x, c.y)
	}
	c.v.AbsMoveCursor(c.x+1, c.y, false)
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
	x, y int
	p    []rune // word thats been written
}

func NewWriteCmd(v *View, x, y int, fchar rune) *WriteCmd {
	return &WriteCmd{v: v, x: x, y: y, p: []rune{fchar}}
}

func (c *WriteCmd) Execute() {
	for i := len(c.p) - 1; i >= 0; i-- {
		c.v.absWriteRune(c.x, c.y, c.p[i])
	}
	c.v.AbsMoveCursor(c.x+len(c.p), c.y, false)
}

func (c *WriteCmd) Reverse() {
	for i := 0; i < len(c.p); i++ {
		c.v.absDeleteRune(c.x, c.y)
	}
	c.v.AbsMoveCursor(c.x, c.y, false)
}

func (c *WriteCmd) Info() string {
	return "Write : " + string(c.p)
}

func (c *WriteCmd) merge(m Mergeable) {
	if o, ok := m.(*WriteCmd); ok {
		c.p = append(c.p, o.p...)
	}
}
