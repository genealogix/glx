package main

import "testing"

func TestVersionString(t *testing.T) {
	origVersion, origCommit, origDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = origVersion, origCommit, origDate
	})

	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{"version only", "1.0.0", "", "", "1.0.0"},
		{"version+commit", "1.0.0", "abc1234def5678", "", "1.0.0 (abc1234)"},
		{"version+commit+date", "1.0.0", "abc1234def5678", "2026-03-30", "1.0.0 (abc1234) 2026-03-30"},
		{"short commit", "1.0.0", "abc", "", "1.0.0 (abc)"},
		{"dev default", "dev", "", "", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, commit, date = tt.version, tt.commit, tt.date
			if got := versionString(); got != tt.want {
				t.Errorf("versionString() = %q, want %q", got, tt.want)
			}
		})
	}
}
