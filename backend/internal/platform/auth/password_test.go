package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("uma-senha-forte")
	if err != nil {
		t.Fatalf("erro ao gerar hash: %v", err)
	}
	ok, err := VerifyPassword("uma-senha-forte", hash)
	if err != nil {
		t.Fatalf("erro ao verificar: %v", err)
	}
	if !ok {
		t.Fatal("senha correta rejeitada")
	}

	ok, err = VerifyPassword("senha-errada", hash)
	if err != nil {
		t.Fatalf("erro ao verificar: %v", err)
	}
	if ok {
		t.Fatal("senha errada foi aceita")
	}
}
