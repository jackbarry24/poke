package core

import (
	"os"
	"regexp"
	"strings"

	"poke/types"

	"github.com/joho/godotenv"
)

type TemplateEngine interface {
	Apply(input string, context map[string]string) string
	ApplyRequest(req *types.PokeRequest, context map[string]string)
	LoadEnv()
}

type DefaultTemplateEngineImpl struct{}

var templatePattern = regexp.MustCompile(`\{\{\s*(.*?)\s*\}\}`)

func (t *DefaultTemplateEngineImpl) LoadEnv() {
	_ = godotenv.Load()
}

func (t *DefaultTemplateEngineImpl) Apply(input string, context map[string]string) string {
	return templatePattern.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))

		// Environment variable: {{ env.VAR_NAME }}
		if strings.HasPrefix(key, "env.") {
			envKey := strings.TrimPrefix(key, "env.")
			if val := os.Getenv(envKey); val != "" {
				return val
			}
			return ""
		}

		// Context key: {{ myKey }}
		if val, ok := context[key]; ok {
			return val
		}
		return ""
	})
}

func (t *DefaultTemplateEngineImpl) ApplyRequest(req *types.PokeRequest, context map[string]string) {
	req.URL = t.Apply(req.URL, context)
	req.Body = t.Apply(req.Body, context)

	for k, v := range req.Headers {
		req.Headers[k] = t.Apply(v, context)
	}
}
