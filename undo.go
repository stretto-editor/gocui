package gocui

import (
	"fmt"
	"reflect"
)

// Should be implemented by every command
type Command interface {
	Info() string
	Execute()
	Reverse()
}

// It should be implemented by a command,
// if 2 successive commands of the same type have to merge
type Mergeable interface {
	merge(m Mergeable)
}

// ActionsInterface should be implemented by our Context
type ActionsInterface interface {
	Exec(c Command)
	Undo()
	Redo()
}

// Implements ActionsInterface
type Context struct {
	merge  bool
	undoSt CmdStack
	redoSt CmdStack
}

// Is used as a stack of Command
// The required methods are implemented below
type CmdStack []Command

// Adds a command to the stack
func (s *CmdStack) Push(c Command) {
	*s = append(*s, c)
}

// Removes the last added command from the stack and returns it, if there is one
// Returns nil otherwise
func (s *CmdStack) Pop() Command {
	le := len(*s)
	if le < 1 {
		return nil
	}
	ret := (*s)[le-1]
	*s = (*s)[0 : le-1]
	return ret
}

// Empties the stack
func (s *CmdStack) Clear() {
	*s = nil
}

// Makes the last command unmergeable.
// Called when the user moves the cursor, for instance
func (con *Context) Cut() {
	con.merge = false
}

// Adds a command to the undo stack,
// merging it with the last command if possible.
// Clears the redo stack
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

// Moves a command from the undo stack to the redo stack and reverses it.
func (con *Context) Undo() {
	if c := con.undoSt.Pop(); c != nil {
		con.redoSt.Push(c)
		con.merge = false
		c.Reverse()
	}
}

// Moves a command from the redo stack to the undo stack and executes it.
func (con *Context) Redo() {
	if c := con.redoSt.Pop(); c != nil {
		con.undoSt.Push(c)
		con.merge = false
		c.Execute()
	}
}

// Returns a formatted string representation of the historic
func (con *Context) ToString(w, h int) string {
	s := make([]byte, 2000)
	l := 0

	var le int = len(con.undoSt)
	var offs int = 0
	// pads to stabilize the position of " - UNDO - "
	if le > h/2-1 {
		offs = le - h/2 + 1
	} else if le < 2 {
		for i := 0; i < h/2-1; i++ {
			l += copy(s[l:], "\n")
		}
	} else {
		for i := 0; i < h/2-le; i++ {
			l += copy(s[l:], "\n")
		}
	}

	// prints the most recent commands in the undo stack
	for i := offs; i < le-1; i++ {
		info := con.undoSt[i].Info()
		if len(info) < w {
			l += copy(s[l:], info+"\n")
		} else {
			// changes the last 3 char to dots if the info is too long
			l += copy(s[l:], info[:w-3]+"..."+"\n")
		}
	}
	l += copy(s[l:], "     - UNDO -\n")
	if le > 0 {
		info := con.undoSt[le-1].Info()
		if len(info) < w {
			l += copy(s[l:], info+"\n")
		} else {
			l += copy(s[l:], info[:w-3]+"..."+"\n")
		}
	} else {
		l += copy(s[l:], "\n")
	}
	l += copy(s[l:], "     - REDO -\n")
	le = len(con.redoSt)
	offs = 0
	// prints the most recent commands in the undo stack
	if le > 0 {
		info := con.redoSt[le-1].Info()
		if len(info) < w {
			l += copy(s[l:], info+"\n")
		} else {
			l += copy(s[l:], info[:w-3]+"..."+"\n")
		}
	} else {
		l += copy(s[l:], "\n")
	}
	l += copy(s[l:], "     -      -\n")

	if le > h/2-1 {
		offs = le - h/2 + 1
	}
	for i := len(con.redoSt) - 2; i >= offs; i-- {
		info := con.redoSt[i].Info()
		if len(info) < w {
			l += copy(s[l:], info+"\n")
		} else {
			l += copy(s[l:], info[:w-3]+"..."+"\n")
		}
	}

	return string(s)
}

func (g *Gui) UpdateHistoric() {
	// this should be in stretto
	var vm, vh *View

	vm = g.Workingview()
	vh, _ = g.View("historic")
	w, h := vh.Size()

	vh.Clear()

	fmt.Fprint(vh, vm.Actions.ToString(w, h))
}
