package domain

import "testing"

func TestDefaultReviewProfile(t *testing.T) {
	p := DefaultReviewProfile()
	if p.IncludeRelated {
		t.Errorf("default profile must not include related context")
	}
	if p.Agentic {
		t.Errorf("default profile must not be agentic")
	}
	if p.Execution {
		t.Errorf("default profile must not enable execution")
	}
	if p.Name != "Diff" {
		t.Errorf("default profile name = %q, want %q", p.Name, "Diff")
	}
}

func TestProfileFromPreset(t *testing.T) {
	cases := []struct {
		preset                  string
		ok                      bool
		includeRelated, agentic bool
	}{
		{"diff", true, false, false},
		{"related", true, true, false},
		{"deep", true, false, true},
		{"full", true, true, true},
		{"DEEP", true, false, true}, // case-insensitive
		{"bogus", false, false, false},
		{"", false, false, false},
	}
	for _, tc := range cases {
		got, ok := ProfileFromPreset(tc.preset)
		if ok != tc.ok {
			t.Errorf("ProfileFromPreset(%q) ok = %v, want %v", tc.preset, ok, tc.ok)
			continue
		}
		if ok && (got.IncludeRelated != tc.includeRelated || got.Agentic != tc.agentic) {
			t.Errorf("ProfileFromPreset(%q) = {related:%v agentic:%v}, want {related:%v agentic:%v}",
				tc.preset, got.IncludeRelated, got.Agentic, tc.includeRelated, tc.agentic)
		}
	}
}
