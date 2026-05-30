package server

import "testing"

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantMin int // minimum token string length
	}{
		{name: "default length when zero", length: 0, wantMin: 32},
		{name: "custom length", length: 8, wantMin: 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateToken(tt.length)
			if err != nil {
				t.Fatalf("generateToken() error = %v", err)
			}
			if len(got) < tt.wantMin {
				t.Fatalf("len(token) = %d, want >= %d", len(got), tt.wantMin)
			}
		})
	}
}

func TestIsBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		expected  string
		provided  string
		wantMatch bool // isBearerToken() result
	}{
		{name: "match", expected: "abc123", provided: "Bearer abc123", wantMatch: true},
		{name: "mismatch", expected: "abc123", provided: "Bearer nope", wantMatch: false},
		{name: "empty expected", expected: "", provided: "Bearer abc123", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBearerToken(tt.expected, tt.provided); got != tt.wantMatch {
				t.Fatalf("isBearerToken() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}
