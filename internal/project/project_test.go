package project

import "testing"

func TestDefaultsArePinned(t *testing.T) {
	if len(Defaults) != 2 || !Defaults[0].Pinned || !Defaults[1].Pinned {
		t.Fatalf("expected two pinned defaults: %#v", Defaults)
	}
}
