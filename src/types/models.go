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
	SavePath     string
	Help         bool
}

type PokeRequest struct {
	Method       string
	URL          string
	Headers      map[string]string
	Body         string
	BodyFile     string
	BodyStdin    bool
	CreatedAt    time.Time
	Meta         *Meta
	Repeat       int
	Workers      int
	ExpectStatus int
}

type PokeResponse struct {
	StatusCode  int
	Header      map[string][]string
	Body        []byte
	ContentType string
	Raw         *http.Response
}

type Meta struct {
	Description string
	Tags        []string
}

type BenchmarkResult struct {
	Total     int
	Successes int
	Failures  int
	Durations []time.Duration
}
