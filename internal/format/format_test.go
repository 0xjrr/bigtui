package format

import (
	"errors"
	"strings"
	"testing"
)

func TestBytes(t *testing.T) {
	cases := map[int64]string{
		0:                "0 B",
		999:              "999 B",
		1200:             "1.2 KB",
		1200000:          "1.2 MB",
		1200000000:       "1.2 GB",
		1200000000000:    "1.2 TB",
		1200000000000000: "1.2 PB",
	}
	for input, want := range cases {
		if got := Bytes(input); got != want {
			t.Fatalf("Bytes(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestLabels(t *testing.T) {
	if got := Labels(map[string]string{"team": "data"}); got != "team=data" {
		t.Fatalf("Labels = %q, want %q", got, "team=data")
	}
	got := Labels(map[string]string{"team": "data", "env": "prod"})
	for _, want := range []string{"team=data", "env=prod"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Labels = %q, missing %q", got, want)
		}
	}
	if Labels(nil) != "" {
		t.Fatalf("Labels(nil) = %q, want empty", Labels(nil))
	}
}

func TestValidationErrorRemovesGoogleAPI400Prefix(t *testing.T) {
	message := ValidationError(errors.New("googleapi: Error 400: Syntax error: Unexpected end of script at [1:1]"))
	if message != "Syntax error: Unexpected end of script at [1:1]" {
		t.Fatalf("unexpected validation error: %q", message)
	}
}

func TestValidationErrorTrimsOtherMessages(t *testing.T) {
	if got := ValidationError(errors.New("  permission denied  ")); got != "permission denied" {
		t.Fatalf("unexpected validation error: %q", got)
	}
}
