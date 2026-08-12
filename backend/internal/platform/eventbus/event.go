// Package eventbus é o barramento de eventos in-process com durabilidade via
// outbox (01-arquitetura.md §5.2): um módulo grava um fato no outbox na
// mesma transação do estado que o gerou; o Relay lê o que ainda não foi
// publicado e despacha pro Bus; o Bus entrega a cada assinante e usa
// processed_event pra nunca rodar o mesmo efeito duas vezes, mesmo que o
// outbox seja relido (entrega é pelo menos uma vez).
package eventbus

import (
	"encoding/json"
	"time"
)

// Event é um fato no passado (01-arquitetura.md §5.3): carrega ID + dados
// mínimos, nunca a entidade inteira — quem consome busca o que precisa.
type Event struct {
	ID          int64
	Aggregate   string
	AggregateID string
	Type        string
	Payload     json.RawMessage
	OccurredAt  time.Time
}
