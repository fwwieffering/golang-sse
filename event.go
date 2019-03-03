package sse

import "fmt"

// Event is the event content for a SSE
type Event struct {
	ID   string
	Data string
}

// FormatEvent returns the event content as something that fits the SSE standard
func (e *Event) FormatEvent() string {
	eventData := ""
	if len(e.ID) > 0 {
		eventData += fmt.Sprintf("id: %s\n", e.ID)
	}
	eventData += fmt.Sprintf("data: %s\n\n", e.Data)
	return eventData
}
