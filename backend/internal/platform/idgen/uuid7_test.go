package idgen

import (
	"regexp"
	"testing"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewUUIDv7Format(t *testing.T) {
	id, err := NewUUIDv7()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !uuidPattern.MatchString(id) {
		t.Fatalf("formato inválido: %s", id)
	}
}

func TestNewUUIDv7Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id, err := NewUUIDv7()
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if seen[id] {
			t.Fatalf("id repetido: %s", id)
		}
		seen[id] = true
	}
}
