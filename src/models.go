package main

import "time"

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
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	BodyFile     string            `json:"body_file,omitempty"`
	BodyStdin    bool              `json:"body_stdin,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	Meta         *Meta             `json:"meta,omitempty"`
	Repeat       int               `json:"repeat,omitempty"`
	Workers      int               `json:"workers,omitempty"`
	ExpectStatus int               `json:"expect_status,omitempty"`
}

type Meta struct {
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
