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

// PromptAssignment links a Prompt to a specific scope in the hierarchy:
// "global" (fallback), "client" (per Git provider), or "project" (per repo).
type PromptAssignment struct {
	ID         int64  `json:"id"`
	PromptID   int64  `json:"prompt_id"`
	PromptName string `json:"prompt_name,omitempty"`
	Scope      string `json:"scope"`
	ClientName string `json:"client_name"`
	ProjectRef string `json:"project_ref"`
}
