package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func RunRequest(req *PokeRequest, verbose bool) {
	if req.Repeat > 1 {
		if req.Workers > req.Repeat {
			req.Workers = req.Repeat
		}
		RunBenchmark(req, verbose)
		return
	}

	// Execute a single request.
	start := time.Now()
	resp, err := SendRequest(*req)
	duration := time.Since(start)
	if err != nil {
		Error("Request failed", err)
	}
	defer resp.Body.Close()
	if req.ExpectStatus != 0 && resp.StatusCode != req.ExpectStatus {
		Error("Unexpected status code", fmt.Errorf("expected %d, got %d", req.ExpectStatus, resp.StatusCode))
	}
	bodyBytes := readResponse(resp)
	if verbose {
		printResponseVerbose(resp, req, bodyBytes, duration)
	} else {
		fmt.Printf("%s\n\n", colorStatus(resp.StatusCode))
		printBody(bodyBytes, resp.Header.Get("Content-Type"))
	}
}

func SendRequest(req PokeRequest) (*http.Response, error) {
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

func HandleSendCommand(sendArg string, opts *CLIOptions) {
	if strings.HasSuffix(sendArg, ".json") {
		// Single JSON file – load and run that request.
		reqPath := resolveRequestPath(sendArg)
		if _, err := os.Stat(reqPath); os.IsNotExist(err) {
			Error(fmt.Sprintf("File '%s' does not exist", reqPath), err)
		}
		loaded, err := loadRequest(reqPath)
		loaded.Body = resolvePayload(loaded.Body, loaded.BodyFile, loaded.BodyStdin, opts.Editor)
		if err != nil {
			Error("Failed to load request from file", err)
		}
		RunRequest(loaded, opts.Verbose)
	} else {
		// Not a JSON file – treat it as a collection.
		filepaths := resolveCollectionFilePaths(sendArg)
		if len(filepaths) > 1 {
			sendCollection(filepaths, opts.Verbose)
		} else if len(filepaths) == 1 {
			loaded, err := loadRequest(filepaths[0])
			if err != nil {
				Error("Failed to load request from file", err)
			}
			RunRequest(loaded, opts.Verbose)
		} else {
			Error(fmt.Sprintf("No JSON files found in %s", sendArg), nil)
		}
	}
}
