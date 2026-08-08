// Package domain contém as regras puras do módulo Metas: entidades, VOs e
// cálculos. Zero dependência externa e zero I/O (01-arquitetura.md §3.1) —
// é o que torna essa camada testável sem banco.
package domain

// RuleError carrega, quando existir, a regra de negócio violada (ex: "RN-02"),
// para a camada HTTP montar o campo `rule` do Problem (RFC 9457, 03-api.md §5)
// sem parsear mensagem em português.
type RuleError struct {
	Rule string
	Msg  string
}

func (e *RuleError) Error() string { return e.Msg }

// NewRuleError também é usado por app/ para erros de validação que dependem
// de estado fora do domínio (ex: "pack não encontrado"), mantendo um único
// tipo de erro que a camada HTTP sabe mapear para Problem.rule.
func NewRuleError(rule, msg string) error {
	return &RuleError{Rule: rule, Msg: msg}
}

func newRuleError(rule, msg string) error {
	return NewRuleError(rule, msg)
}
