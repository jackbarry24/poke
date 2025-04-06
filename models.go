package main

import "time"

type SavedRequest struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	Meta      *Meta             `json:"meta,omitempty"`
}

type Meta struct {
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
