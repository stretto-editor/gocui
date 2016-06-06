// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocui

const maxInt = int(^uint(0) >> 1)

// Editor interface must be satisfied by gocui editors.
type Editor interface {
	Edit(v *View, key Key, ch rune, mod Modifier)
}

// The EditorFunc type is an adapter to allow the use of ordinary functions as
// Editors. If f is a function with the appropriate signature, EditorFunc(f)
// is an Editor object that calls f.
type EditorFunc func(v *View, key Key, ch rune, mod Modifier)

// Edit calls f(v, key, ch, mod)
func (f EditorFunc) Edit(v *View, key Key, ch rune, mod Modifier) {
	f(v, key, ch, mod)
}

// DefaultEditor is the default editor.
var DefaultEditor Editor = EditorFunc(simpleEditor)

// simpleEditor is used as the default gocui editor.
func simpleEditor(v *View, key Key, ch rune, mod Modifier) {
	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
	case key == KeySpace:
		v.EditWrite(' ')
	case key == KeyBackspace || key == KeyBackspace2:
		v.EditDelete(true)
	case key == KeyDelete:
		v.EditDelete(false)
	case key == KeyInsert:
		v.Overwrite = !v.Overwrite
	}
}

// EditWrite writes a rune at the cursor position.
func (v *View) EditWrite(ch rune) {
	v.writeRune(v.cx, v.cy, ch)
	v.MoveCursor(1, 0, true)
}

// EditNewLine inserts a new line under the cursor.
func (v *View) EditNewLine() {
	v.breakLine(v.cx, v.cy)

	y := v.oy + v.cy
	if y >= len(v.viewLines) || (y >= 0 && y < len(v.viewLines) &&
		!(v.Wrap && v.cx == 0 && v.viewLines[y].linesX > 0)) {
		// new line at the end of the buffer or
		// cursor is not at the beginning of a wrapped line
		v.ox = 0
		v.cx = 0
		v.MoveCursor(0, 1, true)
	}
}

// EditDelete deletes a rune at the cursor position. back determines the
// direction.
func (v *View) EditDelete(back bool) {
	x, y := v.ox+v.cx, v.oy+v.cy
	if y < 0 {
		return
	} else if y >= len(v.viewLines) {
		v.MoveCursor(-1, 0, true)
		return
	}

	maxX, _ := v.Size()
	if back {
		if x == 0 { // start of the line
			if y < 1 {
				return
			}

			var maxPrevWidth int
			if v.Wrap {
				maxPrevWidth = maxX
			} else {
				maxPrevWidth = maxInt
			}

			if v.viewLines[y].linesX == 0 { // regular line
				v.mergeLines(v.cy - 1)
				if len(v.viewLines[y-1].line) < maxPrevWidth {
					v.MoveCursor(-1, 0, true)
				}
			} else { // wrapped line
				v.deleteRune(len(v.viewLines[y-1].line)-1, v.cy-1)
				v.MoveCursor(-1, 0, true)
			}
		} else { // middle/end of the line
			v.deleteRune(v.cx-1, v.cy)
			v.MoveCursor(-1, 0, true)
		}
	} else {
		if x == len(v.viewLines[y].line) { // end of the line
			v.mergeLines(v.cy)
		} else { // start/middle of the line
			v.deleteRune(v.cx, v.cy)
		}
	}
}

// isEmpty checks if the view has no line yet
func (v *View) isEmpty() bool {
	return v.lines == nil
}

// bol : beginning of line
func (v *View) bol() bool {
	return v.ox == 0 && v.cx == 0
}

// bob : beginning of buffer
func (v *View) bob() bool {
	return v.cx == 0
}

// eol : end of line
func (v *View) eol() bool {
	rx, ry, err := v.realPosition(v.cx, v.cy)
	if err != nil {
		return false
	}
	return rx == len(v.lines[ry])
}

// eob : end of line in the buffer
// warning : lines in a buffer does not end with '\0' or '\n'
func (v *View) eob() bool {
	return v.cx+v.ox == len(v.viewLines[v.oy].line)
}

// eov : end of view
func (v *View) eov() bool {
	return v.cx+1 == v.x1-v.x0-1
}

// firstLine checks if the current cursor is placed in the first line of the file
func (v *View) firstLine() bool {
	return v.cy+v.oy == 0
}

// lastLine checks if the current cursor is placed in the lastLine line of the file
func (v *View) lastLine() bool {
	_, ry, err := v.realPosition(v.cx, v.cy)
	if err != nil {
		return false
	}
	return ry+1 == len(v.lines)
}

// firstBufferLine checks if the current cursor is placed in the first line of the buffer
func (v *View) firstBufferLine() bool {
	return v.cy == 0
}

// lastBufferLine checks if the current cursor is placed in the lastLine line of the buffer
func (v *View) lastBufferLine() bool {
	return v.cy+1 == v.y1-v.y0-1
}

// lastValidateLineInView checks if the current cursor is placed in the
// last validated line in the view. If the view is full, this corresponds to
// check if the cursor is placed in the last line of the view. If the the text
// does not use all the view, this corresponds to check if the cursor is placed
// in the last line representing the file
func (v *View) lastValidateLineInView() bool {
	return v.cy == len(v.viewLines)
}

// adjustPositionToCurrentString move the cursor and the origin of the view given
// the length of the string to suit to.
func (v *View) adjustPositionToCurrentString() {
	vx := v.ox + v.cx
	vy := v.oy + v.cy
	if vx < 0 || vy < 0 {
		return
	}

	if len(v.viewLines) == 0 {
		// return vx, vy, nil
		return
	}
	if vy < len(v.viewLines) {
		vline := v.viewLines[vy]
		if vx > len(vline.line) {
			v.goToEndOfLine(len(vline.line))
		}
	}
}

// goToEndOfLine will place the cursor at the end of the line given the length
// of the line to feet
func (v *View) goToEndOfLine(lineLength int) {
	maxX, _ := v.Size()
	if lineLength-v.ox < maxX {
		if lineLength < v.ox {
			v.ox = lineLength
		}
		v.cx = lineLength - v.ox
	} else {
		v.ox = lineLength - maxX + 1
		v.cx = maxX
	}
}

// getPreviousLineLength will return the length of the previous line. The current
// cursor's position need to be contain into the view area.
// Take into account wrap and side effect for the first line of the view
func (v *View) getPreviousLineLength() (prevLineWidth int) {
	maxX, _ := v.Size()
	if v.firstBufferLine() {
		_, ry, err := v.realPosition(v.cx, v.cy)
		if err != nil {
			return
		}
		line := v.lines[ry]
		prevLineWidth = len(line) % maxX
	} else {
		prevLineWidth = len(v.viewLines[v.oy+v.cy-1].line)
	}
	return
}

// moveOneRuneForward will move the cursor one character forward and adjust the
// origin of the view if necessary.
func (v *View) moveOneRuneForward(writeMode bool) {
	if v.isEmpty() || (v.lastLine() && v.eol()) {
		return
	}
	if v.eol() && writeMode {
		v.cx++
	} else if v.eol() && !writeMode || v.eov() && v.Wrap {
		_, oy := v.Origin()
		if v.lastBufferLine() {
			v.SetOrigin(0, oy+1)
		} else {
			v.SetOrigin(0, oy)
		}
		if v.eov() && v.Wrap {
			v.cx = 1
		} else {
			v.cx = 0
		}
		v.cy++
	} else if v.eov() && !v.Wrap {
		v.ox++
	} else {
		v.cx++
	}
}

// moveOneRuneBackward will move the cursor one character backward and adjust the
// origin of the view if necessary
func (v *View) moveOneRuneBackward() {
	if v.firstLine() && v.bol() {
		return
	}
	if v.bol() {
		if v.firstBufferLine() {
			v.oy--
		}
		if v.cy != 0 {
			v.cy--
		}

		vline := v.viewLines[v.oy+v.cy]
		len := len(vline.line)
		if v.Wrap {
			v.cx = len
		} else {
			v.goToEndOfLine(len)
		}
	} else if v.bob() {
		v.ox--
	} else {
		v.cx--
	}
	return
}

// moveOneRuneForward will move the cursor one line upper and adjust the
// origin of the view if necessary
func (v *View) moveOneLineUpper() {
	if v.firstLine() {
		return
	}
	if v.firstBufferLine() {
		v.oy--
	} else {
		v.cy--
	}
	return
}

// moveOneRuneForward will move the cursor one line lower and adjust the
// origin of the view if necessary
func (v *View) moveOneLineLower(writeMode bool) {
	if v.isEmpty() || v.lastLine() || (v.lastValidateLineInView() && !writeMode) {
		return
	}
	if v.lastValidateLineInView() && writeMode {
		v.oy++
	} else if v.lastBufferLine() {
		v.oy++
	} else {
		v.cy++
	}
	return
}

// MoveCursor moves the cursor taking into account the width of the line/view,
// displacing the origin if necessary.
func (v *View) MoveCursor(dx, dy int, writeMode bool) {
	if dy < 0 {
		for i := 0; i > dy; i-- {
			v.moveOneLineUpper()
		}
	} else if dy > 0 {
		for i := 0; i < dy; i++ {
			v.moveOneLineLower(writeMode)
		}
	}
	if !writeMode {
		v.adjustPositionToCurrentString()
	}

	// Run through columns
	if dx < 0 {
		for i := 0; i > dx; i-- {
			v.moveOneRuneBackward()
			v.adjustPositionToCurrentString()
		}
	} else if dx > 0 {
		for i := 0; i < dx; i++ {
			v.moveOneRuneForward(writeMode)
		}
	}
}
