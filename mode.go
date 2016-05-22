package gocui

type Mode struct {
	name        string
	keybindings kbSet
}

// CreateMode makes a new kbSet for the mode call name
// If the mode already exists, the kbSet will be flush
func CreateMode(name string) *Mode {
	return &Mode{name: name}
}

func (m *Mode) GetKeyBindings() *kbSet {
	return &m.keybindings
}
