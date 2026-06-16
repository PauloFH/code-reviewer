package domain

import (
	"strings"
	"time"
)

// ReviewProfile descreve quais camadas de contexto a revisão usa.
// A base (diff) está sempre presente; IncludeRelated e Agentic são aditivas.
type ReviewProfile struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	IncludeRelated bool      `json:"include_related"` // issues + MRs relacionados + discussão
	Agentic        bool      `json:"agentic"`         // checkout + navegação por tool-use
	Execution      bool      `json:"execution"`       // sandbox run (Fase 2)
	MaxIterations  int       `json:"max_iterations"`  // teto do loop agêntico (0 = default do env)
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReviewProfileAssignment vincula um ReviewProfile a um escopo na hierarquia:
// "global" (fallback), "client" (por provider Git) ou "project" (por repo).
// Espelha PromptAssignment / AIConfigAssignment.
type ReviewProfileAssignment struct {
	ID          int64  `json:"id"`
	ProfileID   int64  `json:"profile_id"`
	ProfileName string `json:"profile_name,omitempty"`
	Scope       string `json:"scope"`
	ClientName  string `json:"client_name"`
	ProjectRef  string `json:"project_ref"`
}

// DefaultReviewProfile é o perfil efetivo quando nenhum perfil é resolvido por
// escopo: somente diff, preservando o comportamento legado.
func DefaultReviewProfile() ReviewProfile {
	return ReviewProfile{Name: "Diff"}
}

// ProfileFromPreset maps a /review command preset to the corresponding layered
// profile. The second return value is false for unknown presets. The empty
// string is not a preset (it means "no override").
func ProfileFromPreset(preset string) (ReviewProfile, bool) {
	switch strings.ToLower(preset) {
	case "diff":
		return ReviewProfile{Name: "diff"}, true
	case "related":
		return ReviewProfile{Name: "related", IncludeRelated: true}, true
	case "deep":
		return ReviewProfile{Name: "deep", Agentic: true}, true
	case "full":
		return ReviewProfile{Name: "full", IncludeRelated: true, Agentic: true}, true
	default:
		return ReviewProfile{}, false
	}
}
