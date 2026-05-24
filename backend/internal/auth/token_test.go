package auth

import "testing"

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantMin int
	}{
		{name: "default length when zero", length: 0, wantMin: 32},
		{name: "custom length", length: 8, wantMin: 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateToken(tt.length)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}
			if len(got) < tt.wantMin {
				t.Fatalf("len(token) = %d, want >= %d", len(got), tt.wantMin)
			}
		})
	}
}

func TestIsBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		provided string
		want     bool
	}{
		{name: "match", expected: "abc123", provided: "Bearer abc123", want: true},
		{name: "mismatch", expected: "abc123", provided: "Bearer nope", want: false},
		{name: "empty expected", expected: "", provided: "Bearer abc123", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBearerToken(tt.expected, tt.provided); got != tt.want {
				t.Fatalf("IsBearerToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
