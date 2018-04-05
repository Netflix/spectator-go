package spectator

import "testing"

func TestId_Hash(t *testing.T) {
	id := newId("foo", nil)
	h := id.Hash()
	if h == 0 {
		t.Error("Expected a non-zero value, got 0")
	}

	reusesHash := Id{"foo", nil, 1}
	h2 := reusesHash.Hash()
	if h2 != 1 {
		t.Error("Expected 1, got ", h2)
	}
}
