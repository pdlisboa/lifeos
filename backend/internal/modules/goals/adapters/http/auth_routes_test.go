package http

import "testing"

// TestAllRoutesRequireAuth confirma que nenhum endpoint da Fatia 1 responde
// sem autenticação — os IDs no path são placeholders: o handler devolve 401
// antes mesmo de tentar resolver goalId/actionId/etc.
func TestAllRoutesRequireAuth(t *testing.T) {
	ts := newTestServer(t)
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/today"},
		{"GET", "/goals/"},
		{"POST", "/goals/"},
		{"GET", "/goals/x/"},
		{"PATCH", "/goals/x/"},
		{"POST", "/goals/x/activate"},
		{"GET", "/goals/x/probe"},
		{"POST", "/goals/x/probe/answer"},
		{"POST", "/goals/x/probe/skip"},
		{"GET", "/goals/x/delta"},
		{"GET", "/goals/x/consistency"},
		{"GET", "/goals/x/track"},
		{"GET", "/goals/x/action"},
		{"POST", "/goals/x/action"},
		{"GET", "/goals/x/sessions"},
		{"POST", "/goals/x/sessions"},
		{"GET", "/goals/x/evidence"},
		{"POST", "/goals/x/evidence"},
		{"PUT", "/goals/x/competencies/y/level"},
		{"GET", "/goals/x/competencies/y/history"},
		{"POST", "/actions/x/complete"},
		{"POST", "/actions/x/skip"},
		{"GET", "/evidence/x"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			rec := ts.doNoAuth(t, rt.method, rt.path)
			if rec.Code != 401 {
				t.Fatalf("status = %d, want 401 sem autenticação", rec.Code)
			}
		})
	}
}
