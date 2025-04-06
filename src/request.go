package main

import (
	"bytes"
	"net/http"
)

func SendRequest(req SavedRequest) (*http.Response, error) {
	client := &http.Client{}
	request, err := http.NewRequest(req.Method, req.URL, bytes.NewBufferString(req.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		request.Header.Set(k, v)
	}

	return client.Do(request)
}
