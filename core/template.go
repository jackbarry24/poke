package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"poke/types"

	"github.com/Masterminds/sprig/v3"
	"github.com/joho/godotenv"
)

type TemplateContext struct {
	Env     map[string]string
	History map[string]any
}

type TemplateEngine interface {
	LoadEnv()
	LoadHistory() error
	RenderRequest(path string) (*types.PokeRequest, error)
}

type TemplateEngineImpl struct {
	ctx TemplateContext
}

func (t *TemplateEngineImpl) LoadEnv() {
	if t.ctx.Env != nil {
		return
	}
	envMap, err := godotenv.Read(".env")
	if err != nil {
		envMap = map[string]string{}
	}

	for _, kv := range os.Environ() {
		parts := bytes.SplitN([]byte(kv), []byte("="), 2)
		if len(parts) == 2 {
			envMap[string(parts[0])] = string(parts[1])
		}
	}
	t.ctx.Env = envMap
}
func (t *TemplateEngineImpl) LoadHistory() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	pokePath := filepath.Join(homeDir, ".poke", "tmp_poke_latest.json")
	data, err := os.ReadFile(pokePath)
	if err != nil {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if bodyStr, ok := raw["body"].(string); ok {
		var parsed any
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err == nil {
			raw["body"] = parsed
		}
	}

	t.ctx.History = raw
	return nil
}

func (t *TemplateEngineImpl) RenderRequest(data []byte) (*types.PokeRequest, error) {
	t.LoadEnv()
	if err := t.LoadHistory(); err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}

	tmpl, err := template.New("poke").
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{
			// env: returns the environment map for property-style lookup, e.g., {{ env.TOKEN }}
			"env": func() map[string]string {
				return t.ctx.Env
			},
			// history: returns the history map for property-style lookup, e.g., {{ history.body }} or nested {{ history.body.data }}
			"history": func() any {
				return t.ctx.History
			},
		}).
		Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("template parse: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, t.ctx)
	if err != nil {
		return nil, fmt.Errorf("template exec: %w", err)
	}

	var req types.PokeRequest
	if err := json.Unmarshal(buf.Bytes(), &req); err != nil {
		return nil, fmt.Errorf("unmarshal templated request: %w", err)
	}

	return &req, nil
}
