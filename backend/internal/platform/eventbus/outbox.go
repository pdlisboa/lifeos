package eventbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/phablo/lifeos/internal/platform/db"
)

// Append grava um evento no outbox. Chame dentro da mesma transação
// (pgx.Tx) que grava o estado que o evento descreve — é essa atomicidade
// que garante que evento e estado nunca divergem (01-arquitetura.md §5.2).
// Também aceita *pgxpool.Pool direto, para os poucos casos sem transação.
func Append(ctx context.Context, tx db.TX, aggregate, aggregateID, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("eventbus: serializar payload de %s: %w", eventType, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox (aggregate, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4)`,
		aggregate, aggregateID, eventType, body,
	)
	if err != nil {
		return fmt.Errorf("eventbus: inserir outbox %s: %w", eventType, err)
	}
	return nil
}
