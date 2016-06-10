package gocui

import "strings"

func (v *View) SearchForward(pattern string) (bool, int, int) {
	rx, ry, _ := v.realPosition(v.cx, v.cy)
	if len(pattern) == 0 {
		pattern = v.searchString
	}
	if len(pattern) > 0 {
		v.searchString = pattern

		// Start searching one character beyond where we are
		// or we won't be able to continue to the next match
		if len(v.lines[ry]) > rx+1 {
			if ind := strings.Index(string(v.lines[ry][rx+1:]), pattern); ind > -1 {
				return true, ind + rx, ry
			}
		}
		for i := ry + 1; i < len(v.lines); i++ {
			if ind := strings.Index(string(v.lines[i]), pattern); ind > -1 {
				return true, ind, i
			}
		}
	}
	return false, 0, 0
}
