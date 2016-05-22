package gocui

type Mode struct {
	name        string
	keybindings kbSet
}

func CreateMode(name string) *Mode {
	return &Mode{name: name}
}

func (m *Mode) GetKeyBindings() *kbSet {
	return &m.keybindings
}

func (m *Mode) Name() string {
	return m.name
}
