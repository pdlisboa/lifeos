package agents

import (
	"encoding/json"
	"fmt"
)

// schemaNode é o subconjunto conservador de JSON Schema que os prompts deste
// projeto usam (04-agentes.md §4.8: os 5 schemas do projeto são
// deliberadamente conservadores — sem oneOf/$defs/$ref, porque o modo
// `strict` do provedor não é garantia pra construções exóticas). Cobre:
// type, properties, required, additionalProperties, items,
// minLength/maxLength, minimum/maximum, enum, minItems/maxItems.
// schemaType aceita tanto `"type": "string"` quanto `"type": ["string",
// "null"]` — a segunda forma é como os 5 schemas do projeto marcam campo
// opcional em modo `strict` (nenhum usa `oneOf`/`$ref` de propósito,
// 04-agentes.md §4.8, mas `type` com múltiplos valores é JSON Schema comum
// e vários provedores aceitam bem em `strict`).
type schemaType []string

func (t *schemaType) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*t = schemaType{single}
		return nil
	}
	var multi []string
	if err := json.Unmarshal(data, &multi); err != nil {
		return err
	}
	*t = schemaType(multi)
	return nil
}

func (t schemaType) has(kind string) bool {
	for _, k := range t {
		if k == kind {
			return true
		}
	}
	return false
}

type schemaNode struct {
	Type                 schemaType            `json:"type"`
	Properties           map[string]schemaNode `json:"properties"`
	Required             []string              `json:"required"`
	AdditionalProperties *bool                 `json:"additionalProperties"`
	Items                *schemaNode           `json:"items"`
	MinLength            *int                  `json:"minLength"`
	MaxLength            *int                  `json:"maxLength"`
	Minimum              *float64              `json:"minimum"`
	Maximum              *float64              `json:"maximum"`
	Enum                 []any                 `json:"enum"`
	MinItems             *int                  `json:"minItems"`
	MaxItems             *int                  `json:"maxItems"`
}

// validate roda a saída do agente contra o schema completo — estrutura E
// restrições de valor (01-arquitetura.md §6.4). É a garantia real, não uma
// segunda camada: o `strict` do provedor tipicamente ignora minLength,
// minimum/maximum, enum etc (04-agentes.md §4.8), e é justamente nessas
// restrições que moram as regras de negócio do produto.
func validate(schemaBytes, data []byte) []string {
	var schema schemaNode
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return []string{fmt.Sprintf("schema inválido: %v", err)}
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return []string{fmt.Sprintf("saída não é JSON válido: %v", err)}
	}

	var errs []string
	walk("$", schema, value, &errs)
	return errs
}

func walk(path string, schema schemaNode, value any, errs *[]string) {
	if len(schema.Type) == 0 {
		return // sem 'type' declarado, nada a checar nesse nó
	}
	if value == nil {
		if schema.Type.has("null") {
			return
		}
		*errs = append(*errs, path+": valor null não permitido")
		return
	}
	// campo nullable (ex.: `["string","null"]`): o valor não-null precisa
	// bater com o outro tipo declarado, que é sempre único na prática — os
	// schemas do projeto nunca combinam dois tipos concretos, só um tipo +
	// "null" (04-agentes.md §4.8).
	for _, t := range schema.Type {
		if t == "null" {
			continue
		}
		switch t {
		case "object":
			walkObject(path, schema, value, errs)
		case "array":
			walkArray(path, schema, value, errs)
		case "string":
			walkString(path, schema, value, errs)
		case "integer", "number":
			walkNumber(path, schema, value, errs)
		case "boolean":
			if _, ok := value.(bool); !ok {
				*errs = append(*errs, path+": esperava boolean")
			}
		}
		return
	}
}

func walkObject(path string, schema schemaNode, value any, errs *[]string) {
	obj, ok := value.(map[string]any)
	if !ok {
		*errs = append(*errs, path+": esperava object")
		return
	}
	for _, req := range schema.Required {
		if _, ok := obj[req]; !ok {
			*errs = append(*errs, fmt.Sprintf("%s.%s: campo obrigatório ausente", path, req))
		}
	}
	if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
		for k := range obj {
			if _, known := schema.Properties[k]; !known {
				*errs = append(*errs, fmt.Sprintf("%s.%s: propriedade não declarada no schema", path, k))
			}
		}
	}
	for k, sub := range schema.Properties {
		if v, present := obj[k]; present {
			walk(path+"."+k, sub, v, errs)
		}
	}
}

func walkArray(path string, schema schemaNode, value any, errs *[]string) {
	arr, ok := value.([]any)
	if !ok {
		*errs = append(*errs, path+": esperava array")
		return
	}
	if schema.MinItems != nil && len(arr) < *schema.MinItems {
		*errs = append(*errs, fmt.Sprintf("%s: tem %d itens, mínimo %d", path, len(arr), *schema.MinItems))
	}
	if schema.MaxItems != nil && len(arr) > *schema.MaxItems {
		*errs = append(*errs, fmt.Sprintf("%s: tem %d itens, máximo %d", path, len(arr), *schema.MaxItems))
	}
	if schema.Items != nil {
		for i, item := range arr {
			walk(fmt.Sprintf("%s[%d]", path, i), *schema.Items, item, errs)
		}
	}
}

func walkString(path string, schema schemaNode, value any, errs *[]string) {
	s, ok := value.(string)
	if !ok {
		*errs = append(*errs, path+": esperava string")
		return
	}
	if schema.MinLength != nil && len(s) < *schema.MinLength {
		*errs = append(*errs, fmt.Sprintf("%s: tem %d caracteres, mínimo %d", path, len(s), *schema.MinLength))
	}
	if schema.MaxLength != nil && len(s) > *schema.MaxLength {
		*errs = append(*errs, fmt.Sprintf("%s: tem %d caracteres, máximo %d", path, len(s), *schema.MaxLength))
	}
	if len(schema.Enum) > 0 && !enumContains(schema.Enum, s) {
		*errs = append(*errs, path+": valor fora do enum")
	}
}

func walkNumber(path string, schema schemaNode, value any, errs *[]string) {
	n, ok := value.(float64) // encoding/json decodifica number sempre como float64
	if !ok {
		*errs = append(*errs, path+": esperava number")
		return
	}
	if schema.Minimum != nil && n < *schema.Minimum {
		*errs = append(*errs, fmt.Sprintf("%s: %g abaixo do mínimo %g", path, n, *schema.Minimum))
	}
	if schema.Maximum != nil && n > *schema.Maximum {
		*errs = append(*errs, fmt.Sprintf("%s: %g acima do máximo %g", path, n, *schema.Maximum))
	}
}

func enumContains(enum []any, s string) bool {
	for _, v := range enum {
		if sv, ok := v.(string); ok && sv == s {
			return true
		}
	}
	return false
}
