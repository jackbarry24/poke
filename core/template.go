package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"poke/types"
	"poke/util"

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

var templateExpr = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

func (t *DefaultTemplateEngineImpl) LoadHistory() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	pokePath := filepath.Join(homeDir, ".poke", "tmp_poke_latest.json")
	data, err := os.ReadFile(pokePath)
	if err != nil {
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if bodyStr, ok := raw["body"].(string); ok {
		var parsed interface{}
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err == nil {
			raw["body"] = parsed
		}
	}

	t.history = raw
	return nil
}

func (t *DefaultTemplateEngineImpl) getEnvValue(key string) string {
	if t.dotenv == nil {
		envMap, err := godotenv.Read(".env")
		if err != nil {
			envMap = map[string]string{}
		}
		t.dotenv = envMap
	}

	if val, ok := t.dotenv[key]; ok && val != "" {
		return val
	}

	val := os.Getenv(key)
	if val != "" {
		t.dotenv[key] = val
	}
	return val
}

func (t *DefaultTemplateEngineImpl) Apply(input string) string {
	if input == "" {
		return ""
	}

	result := templateExpr.ReplaceAllStringFunc(input, func(match string) string {
		parts := templateExpr.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		key := parts[1]
		if strings.HasPrefix(key, "env.") {
			envKey := strings.TrimPrefix(key, "env.")
			val := t.getEnvValue(envKey)
			util.Debug("template", fmt.Sprintf("Env %s = %q", envKey, val))
			return val
		}
		if strings.HasPrefix(key, "history.") {
			historyKey := strings.TrimPrefix(key, "history.")
			if err := t.LoadHistory(); err != nil {
				util.Debug("template", fmt.Sprintf("History load error: %s", err))
				return match
			}
			parts := strings.Split(historyKey, ".")
			var current interface{} = t.history
			for _, part := range parts {
				if m, ok := current.(map[string]interface{}); ok {
					current = m[part]
				} else {
					return match
				}
			}
			if str, ok := current.(string); ok {
				return str
			} else {
				bytes, err := json.Marshal(current)
				if err == nil {
					return string(bytes)
				}
				return match
			}
		}
		return match
	})

	util.Debug("template", fmt.Sprintf("Rendered: %q -> %q", input, result))
	return result
}

func (t *DefaultTemplateEngineImpl) ApplyToRequest(req *types.PokeRequest) {
	if req == nil {
		return
	}
	req.URL = t.Apply(req.URL)
	req.Body = t.Apply(req.Body)
	req.BodyFile = t.Apply(req.BodyFile)

	newHeaders := make(map[string][]string)
	for k, v := range req.Headers {
		newKey := t.Apply(k)
		newValues := []string{}
		for _, value := range v {
			newValues = append(newValues, t.Apply(value))
		}
		newHeaders[newKey] = newValues
	}
	req.Headers = newHeaders

	newQueryParams := make(map[string][]string)
	for k, v := range req.QueryParams {
		newKey := t.Apply(k)
		newValues := []string{}
		for _, value := range v {
			newValues = append(newValues, t.Apply(value))
		}
		newQueryParams[newKey] = newValues
	}
	req.QueryParams = newQueryParams
}
