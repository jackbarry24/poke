package types

import (
	"net/http"
	"time"
)

type CLIOptions struct {
	Method       string
	Data         string
	DataFile     string
	DataStdin    bool
	UserAgent    string
	Headers      string
	Verbose      bool
	Repeat       int
	Workers      int
	ExpectStatus int
	Retries      int
	Backoff      int
	DryRun       bool
	Editor       bool
	SavePath     string
	Help         bool
}

type PokeResponse struct {
	StatusCode  int                 `json:"status_code"`
	Status      string              `json:"status"`
	Headers     map[string][]string `json:"headers"`
	Body        []byte              `json:"body"`
	ContentType string              `json:"content_type"`
	Raw         *http.Response      `json:"-"` // not serializable
	Timestamp   time.Time           `json:"timestamp"`
	Duration    time.Duration       `json:"duration"`
}

type PokeRequest struct {
	Method      string              `json:"method"`
	FullURL     string              `json:"-"`
	Scheme      string              `json:"scheme"`
	Host        string              `json:"host"`
	Path        string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	QueryParams map[string][]string `json:"query_params"`
	Body        []byte              `json:"body"`
	BodyFile    string              `json:"body_file"`
	Meta        *Meta               `json:"meta"`
	Retries     int                 `json:"retries"`
	Backoff     int                 `josn:"backoff"`
	Repeat      int                 `json:"repeat"`
	Workers     int                 `json:"workers"`
	Assert      *Assertions         `json:"assert"`
}

type Assertions struct {
	Status       int                 `json:"status"`
	BodyContains string              `json:"body_contains"`
	Headers      map[string][]string `json:"headers"`
}

type Meta struct {
	CreatedAt   time.Time
	Description string
	Tags        []string
}

type BenchmarkResult struct {
	Total     int
	Successes int
	Failures  int
	Durations []time.Duration
}
