package gocui

// Mode is the struct that associates to a mode name a set of keybindings
// and functions to execute when you switch to the mode or from the mode to another
type modeHandler func(g *Gui) error

// Mode is the struct that associates to a mode name a set of keybindings
// and functions to execute when you switch to the mode or from the mode to another
type Mode struct {
	name        string
	keybindings kbSet
	openMode    modeHandler
	closeMode   modeHandler
}

//CreateMode create a mode with the given name, and opening and closing functions
func CreateMode(name string, openMode modeHandler, closeMode modeHandler) *Mode {
	return &Mode{name: name, openMode: openMode, closeMode: closeMode}
}

//GetKeyBindings gives a pointer to the set of keybindings associate to the mode
func (m *Mode) GetKeyBindings() *kbSet {
	return &m.keybindings
}

// Name returns the name of the mode
func (m *Mode) Name() string {
	return m.name
}

// OpenMode execute the file handler to execute at the opening of the mode
func (m *Mode) OpenMode(g *Gui) {
	if m.openMode != nil {
		m.openMode(g)
	}
}

// CloseMode execute the file handler to execute at the closing time of the mode
func (m *Mode) CloseMode(g *Gui) {
	if m.closeMode != nil {
		m.closeMode(g)
	}
}
