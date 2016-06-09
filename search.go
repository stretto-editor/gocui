package gocui

import "strings"

func (v *View) SearchForward(pattern string) (bool, int, int) {
	rx, ry, _ := v.realPosition(v.cx, v.cy)
	if len(pattern) > 0 {
		if ind := strings.Index(string(v.lines[ry][rx:]), pattern); ind > -1 {
			return true, ind + rx, ry
		}
		for i := ry + 1; i < len(v.lines); i++ {
			if ind := strings.Index(string(v.lines[i]), pattern); ind > -1 {
				return true, ind, i
			}
		}
	}
	return false, 0, 0
}
