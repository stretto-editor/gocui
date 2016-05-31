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
		//case key == KeyEnter:
		//v.EditNewLine()
		/*
			case key == KeyArrowDown:
				v.MoveCursor(0, 1, false)
			case key == KeyArrowUp:
				v.MoveCursor(0, -1, false)
			case key == KeyArrowLeft:
				v.MoveCursor(-1, 0, false)
			case key == KeyArrowRight:
				v.MoveCursor(1, 0, false)
		*/
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
	return v.cy+1 == len(v.viewLines)
}

// adjustPositionToCurrentString move the cursor and the origin af the view given
// the length of the string to suit to.
func (v *View) adjustPositionToCurrentString(width int) {
	if v.Wrap {
		v.cx = width
	} else {
		maxX, _ := v.Size()
		if width-v.ox < maxX {
			if width < v.ox {
				v.ox = width
			}
			v.cx = width - v.ox
		} else {
			v.ox = width - maxX + 1
			v.cx = maxX
		}
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

// moveOneRuneForward will move the cursor one character backward and adjust the
// origin of the view if necessary
func (v *View) moveOneRuneBackward() {
	if v.firstLine() && v.bol() {
		return
	}
	if v.bol() {
		prevLineLength := v.getPreviousLineLength()
		if v.firstBufferLine() {
			v.oy--
		}
		v.cy--
		v.adjustPositionToCurrentString(prevLineLength)
	} else if v.bob() {
		v.ox--
	} else {
		v.cx--
	}
}

// moveOneRuneForward will move the cursor one line upper and adjust the
// origin of the view if necessary
func (v *View) moveOneLineUpper() {
	if v.firstLine() {
		return
	}
	prevLineLength := v.getPreviousLineLength()
	if v.firstBufferLine() {
		v.oy--
	}
	v.cy--
	if v.cx >= prevLineLength {
		v.adjustPositionToCurrentString(prevLineLength)
	}
}

// moveOneRuneForward will move the cursor one line lower and adjust the
// origin of the view if necessary
func (v *View) moveOneLineLower() {
	if v.isEmpty() || v.lastLine() || v.lastValidateLineInView() {
		return
	}
	var nextLineWidth int
	if v.lastBufferLine() {
		maxX, _ := v.Size()
		v.oy++
		_, ry, err := v.realPosition(v.cx, v.cy)
		if err != nil {
			return
		}
		line := v.lines[ry]
		nextLineWidth = len(line) % maxX
	} else {
		nextLineWidth = len(v.viewLines[v.oy+v.cy+1].line)
		v.cy++
	}
	if v.cx >= nextLineWidth {
		v.adjustPositionToCurrentString(nextLineWidth)
	}
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
			v.moveOneLineLower()
		}
	}

	if dx < 0 {
		for i := 0; i > dx; i-- {
			v.moveOneRuneBackward()
		}
	} else if dx > 0 {
		for i := 0; i < dx; i++ {
			v.moveOneRuneForward(writeMode)
		}
	}
	// if writeMode && v.eol() && dx > 0 {
	// 	v.moveOneRuneForward(writeMode)
	// }
	//
	// maxX, maxY := v.Size()
	// cx, cy := v.cx+dx, v.cy+dy
	// x, y := v.ox+cx, v.oy+cy
	// if y > len(v.lines) {
	// 	cy -= y - len(v.lines)
	// 	y -= y - len(v.lines)
	// }
	//
	// var curLineWidth, prevLineWidth int
	// // get the width of the current line
	// if writeMode {
	// 	if v.Wrap {
	// 		curLineWidth = maxX - 1
	// 	} else {
	// 		curLineWidth = maxInt
	// 	}
	// } else {
	// 	if y >= 0 && y < len(v.viewLines) {
	// 		curLineWidth = len(v.viewLines[y].line)
	// 		if v.Wrap && curLineWidth >= maxX {
	// 			curLineWidth = maxX - 1
	// 		}
	// 	} else {
	// 		curLineWidth = 0
	// 	}
	// }
	// // get the width of the previous line
	// if y-1 >= 0 && y-1 < len(v.viewLines) {
	// 	prevLineWidth = len(v.viewLines[y-1].line)
	// } else {
	// 	prevLineWidth = 0
	// }
	//
	// // adjust cursor's x position and view's x origin
	// if x > curLineWidth { // move to next line
	// 	if dx > 0 { // horizontal movement
	// 		if !v.Wrap {
	// 			v.ox = 0
	// 		}
	// 		v.cx = 0
	// 		cy++
	// 	} else { // vertical movement
	// 		if curLineWidth > 0 { // move cursor to the EOL
	// 			if v.Wrap {
	// 				v.cx = curLineWidth
	// 			} else {
	// 				ncx := curLineWidth - v.ox
	// 				if ncx < 0 {
	// 					v.ox += ncx
	// 					if v.ox < 0 {
	// 						v.ox = 0
	// 					}
	// 					v.cx = 0
	// 				} else {
	// 					v.cx = ncx
	// 				}
	// 			}
	// 		} else {
	// 			if !v.Wrap {
	// 				v.ox = 0
	// 			}
	// 			v.cx = 0
	// 		}
	// 	}
	// } else if cx < 0 {
	// 	if !v.Wrap && v.ox > 0 { // move origin to the left
	// 		v.ox--
	// 	} else { // move to previous line
	// 		if prevLineWidth > 0 {
	// 			if !v.Wrap { // set origin so the EOL is visible
	// 				nox := prevLineWidth - maxX + 1
	// 				if nox < 0 {
	// 					v.ox = 0
	// 				} else {
	// 					v.ox = nox
	// 				}
	// 			}
	// 			v.cx = prevLineWidth
	// 		} else {
	// 			if !v.Wrap {
	// 				v.ox = 0
	// 			}
	// 			v.cx = 0
	// 		}
	// 		cy--
	// 	}
	// } else { // stay on the same line
	// 	if v.Wrap {
	// 		v.cx = cx
	// 	} else {
	// 		if cx >= maxX {
	// 			v.ox++
	// 		} else {
	// 			v.cx = cx
	// 		}
	// 	}
	// }
	//
	// // adjust cursor's y position and view's y origin
	// if cy >= maxY {
	// 	v.oy++
	// } else if cy < 0 {
	// 	if v.oy > 0 {
	// 		v.oy--
	// 	}
	// } else {
	// 	v.cy = cy
	// }
}
