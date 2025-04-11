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
	Editor       bool
	SavePath     string // save to .poke
	Help         bool
}

type PokeRequest struct {
	Method      string
	URL         string
	Headers     map[string]string
	QueryParams map[string]string
	Body        string
	BodyFile    string
	BodyStdin   bool
	Meta        *Meta
	Repeat      int
	Workers     int
	Assert      *Assertions
}

type PokeResponse struct {
	StatusCode  int
	Header      map[string][]string
	Body        []byte
	ContentType string
	Raw         *http.Response
}

type Assertions struct {
	Status       int
	BodyContains string
	Headers      map[string]string
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
