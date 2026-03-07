package pathutil

import "testing"

func TestNormalizeServerURLAllowsPrivateTargets(t *testing.T) {
	t.Parallel()

	tests := []string{
		"http://127.0.0.1:8080",
		"http://localhost:8080",
		"http://10.0.0.5:8080",
		"http://172.16.10.20:8080",
		"http://192.168.1.8:8080",
		"http://100.64.1.2:8080",
	}

	for _, input := range tests {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeServerURL(input)
			if err != nil {
				t.Fatalf("NormalizeServerURL(%q) error = %v", input, err)
			}
			if got != input {
				t.Fatalf("NormalizeServerURL(%q) = %q, want %q", input, got, input)
			}
		})
	}
}

func TestNormalizeServerURLRejectsPublicOrMalformedTargets(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"not-a-url",
		"http://example.com:8080",
		"http://8.8.8.8:8080",
		"http://127.0.0.1",
		"http://user:pass@127.0.0.1:8080",
		"http://127.0.0.1:8080/path",
	}

	for _, input := range tests {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := NormalizeServerURL(input); err == nil {
				t.Fatalf("NormalizeServerURL(%q) error = nil, want non-nil", input)
			}
		})
	}
}
