package domain

import "time"

// Prompt is a reusable review prompt configuration containing
// a system prompt and an output template, both stored as HTML.
type Prompt struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	SystemPrompt   string    `json:"system_prompt"`
	OutputTemplate string    `json:"output_template"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (p Prompt) Validate() error {
	if p.Name == "" {
		return Invalid("name is required")
	}
	return nil
}
