package postgres

// i16ptr/intptr convertem entre os inteiros do domínio (sempre `int`, `*int`)
// e os tipos que o Postgres/sqlc usam para `smallint` (`int16`, `*int16`).
// Existem só por causa dessa largura de coluna — o domínio não deveria
// precisar saber que `current_level` é smallint no banco.
func i16ptr(v *int) *int16 {
	if v == nil {
		return nil
	}
	n := int16(*v)
	return &n
}

func intptr(v *int16) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
