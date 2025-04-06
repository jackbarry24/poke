package main

import (
	"bytes"
	"net/http"
)

func SendRequest(method, url string, headers map[string]string, data string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBufferString(data))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return client.Do(req)
}
