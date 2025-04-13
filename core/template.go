package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"poke/types"

	"github.com/joho/godotenv"
)

type TemplateEngine interface {
	LoadHistory() error
	Apply(input string) string
	ApplyToRequest(req *types.PokeRequest)
}

type DefaultTemplateEngineImpl struct {
	history map[string]interface{}
	dotenv  map[string]string
}

func (t *DefaultTemplateEngineImpl) LoadHistory() error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	pokePath := filepath.Join(homedir, ".poke", "tmp_poke_latest.json")
	file, err := os.Open(pokePath)
	if err != nil {
		return nil // No-op if no history
	}
	defer file.Close()

	var raw map[string]interface{}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return err
	}

	// Unmarshal body if JSON
	if bodyStr, ok := raw["Body"].(string); ok {
		var parsed interface{}
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err == nil {
			raw["Body"] = parsed
		}
	}

	t.history = raw
	return nil
}

var templateExpr = regexp.MustCompile(`{{\s*([a-zA-Z0-9_.]+)\s*}}`)

func (t *DefaultTemplateEngineImpl) Apply(input string) string {
	if input == "" {
		return ""
	}

	ctx := map[string]interface{}{
		"env":     make(map[string]string),
		"history": t.history,
	}

	// Find used env vars and load only those
	matches := templateExpr.FindAllStringSubmatch(input, -1)
	seenEnv := map[string]bool{}
	for _, match := range matches {
		path := match[1]
		if strings.HasPrefix(path, "env.") {
			key := strings.TrimPrefix(path, "env.")
			if _, exists := seenEnv[key]; !exists {
				seenEnv[key] = true
				val := t.getEnvValue(key)
				fmt.Printf("[env] %s = %q\n", key, val)
				ctx["env"].(map[string]string)[key] = val
			}
		}
	}

	fmt.Println("[tmpl] Template context: env =", ctx["env"])

	tmpl, err := template.New("poke").Parse(input)
	if err != nil {
		fmt.Println("[tmpl] Parse error:", err)
		return input
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, ctx)
	if err != nil {
		fmt.Println("[tmpl] Exec error:", err)
		return input
	}

	fmt.Printf("[tmpl] Rendered: %q -> %q\n", input, buf.String())
	return buf.String()
}

func (t *DefaultTemplateEngineImpl) getEnvValue(key string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}

	if t.dotenv == nil {
		t.dotenv, _ = godotenv.Read(".env")
	}
	return t.dotenv[key]
}

func (t *DefaultTemplateEngineImpl) ApplyToRequest(req *types.PokeRequest) {
	if req == nil {
		return
	}

	req.URL = t.Apply(req.URL)
	req.Body = t.Apply(req.Body)
	req.BodyFile = t.Apply(req.BodyFile)

	newHeaders := make(map[string]string)
	for k, v := range req.Headers {
		newHeaders[t.Apply(k)] = t.Apply(v)
	}
	req.Headers = newHeaders

	newQueryParams := make(map[string]string)
	for k, v := range req.QueryParams {
		newQueryParams[t.Apply(k)] = t.Apply(v)
	}
	req.QueryParams = newQueryParams
}
