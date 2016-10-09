package plugin

import "strings"

type UI struct {
	UI
}

func (ui *UI) Confirm(message string) bool {
	response := ui.Ask(message)
	switch strings.ToLower(response) {
	case "y", "yes":
		return true
	}
	return false
}
