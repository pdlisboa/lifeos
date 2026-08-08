// Package httpx tem helpers de resposta HTTP compartilhados por todos os módulos.
package httpx

import (
	"encoding/json"
	"net/http"
)

// Problem segue RFC 9457 (03-api.md §5). rule carrega a regra de negócio
// violada (ex: "RN-02"), quando existir, para a UI tratar sem parsear texto.
type Problem struct {
	Type   string `json:"type,omitempty"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	Rule   string `json:"rule,omitempty"`
}

func WriteProblem(w http.ResponseWriter, status int, title, detail, rule string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{
		Title:  title,
		Status: status,
		Detail: detail,
		Rule:   rule,
	})
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
