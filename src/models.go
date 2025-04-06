package main

import "time"

type PokeRequest struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
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
