package keyboard

import (
	"fmt"
	"strings"
)

// CallbackData represents parsed callback data
type CallbackData struct {
	Action string // "mode", "proj", "skip", "explain", "dl", "action"
	Value  string // The parameter
}

// ParseCallback parses callback data string
func ParseCallback(data string) (*CallbackData, error) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid callback format: %s", data)
	}

	return &CallbackData{
		Action: parts[0],
		Value:  parts[1],
	}, nil
}

// EncodeCallback creates callback data string
func EncodeCallback(action, value string) string {
	return fmt.Sprintf("%s:%s", action, value)
}
