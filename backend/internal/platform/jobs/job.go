// Package jobs é a fila de jobs escrita à mão em cima do Postgres
// (01-arquitetura.md §7): SELECT ... FOR UPDATE SKIP LOCKED, retry com
// backoff, unique_key pra não duplicar trabalho, cancelamento por context.
// Sem Redis, sem River — decisão pedagógica (ADR A3): a fila exercita
// concorrência, transação e context de uma vez, e o código cabe inteiro na
// cabeça de quem mantém.
package jobs

import (
	"context"
	"encoding/json"
)

type Status string

const (
	StatusQueued  Status = "queued"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
	StatusDead    Status = "dead"
)

// Job é o que o Worker entrega ao Handler depois de reivindicar a linha com
// SKIP LOCKED. Attempts já inclui a tentativa atual (o fetch incrementa
// antes de devolver).
type Job struct {
	ID          int64
	Kind        string
	Payload     json.RawMessage
	Attempts    int16
	MaxAttempts int16
}

// Handler processa um Job. Um erro devolvido agenda retry com backoff (ou
// marca 'dead' se esgotou max_attempts). O ctx é cancelado no shutdown do
// worker — handlers de chamada longa (ex.: agente de LLM) devem respeitá-lo.
type Handler func(ctx context.Context, job Job) error
