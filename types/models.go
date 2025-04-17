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
	SavePath     string // save to .poke
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
	Scheme      string              `josn:"scheme"`
	Host        string              `json:"host"`
	Path        string              `json:"Path"`
	Headers     map[string][]string `json:"headers"`
	QueryParams map[string][]string `json:"query_params"`
	Body        string              `json:"body"`
	BodyFile    string              `json:"body_file"`
	BodyStdin   bool                `json:"body_stdin"`
	Meta        *Meta               `json:"meta"`
	Retries     int                 `json:"retries"`
	Repeat      int                 `json:"repeat"`
	Workers     int                 `json:"workers"`
	Assert      *Assertions         `json:"assert"`
}

type Assertions struct {
	Status       int
	BodyContains string
	Headers      map[string][]string
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
