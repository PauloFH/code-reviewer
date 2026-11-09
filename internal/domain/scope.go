package domain

// Scope is where an assignment applies; resolution picks the most specific
// match: project, then client, then global.
type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeClient  Scope = "client"
	ScopeProject Scope = "project"
)

// ScopeKey identifies one slot in the scope hierarchy.
type ScopeKey struct {
	Scope      Scope  `json:"scope"`
	ClientName string `json:"client_name"`
	ProjectRef string `json:"project_ref"`
}

// Normalize clears the fields the scope does not use and rejects missing ones,
// so each slot has exactly one representation.
func (k ScopeKey) Normalize() (ScopeKey, error) {
	switch k.Scope {
	case ScopeGlobal:
		return ScopeKey{Scope: ScopeGlobal}, nil
	case ScopeClient:
		if k.ClientName == "" {
			return k, Invalid("client_name is required for client scope")
		}
		return ScopeKey{Scope: ScopeClient, ClientName: k.ClientName}, nil
	case ScopeProject:
		if k.ClientName == "" || k.ProjectRef == "" {
			return k, Invalid("client_name and project_ref are required for project scope")
		}
		return k, nil
	}
	return k, Invalid("scope must be global, client, or project")
}

// Assignment links a prompt, AI config or review profile (the target) to a scope.
type Assignment struct {
	ID         int64  `json:"id"`
	TargetID   int64  `json:"target_id"`
	TargetName string `json:"target_name,omitempty"`
	ScopeKey
}
