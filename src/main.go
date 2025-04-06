package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	method := flag.String("X", "GET", "HTTP method to use")
	flag.StringVar(method, "method", "GET", "HTTP method to use")

	data := flag.String("d", "", "Request body payload")
	flag.StringVar(data, "data", "", "Request body payload")

	userAgent := flag.String("A", "poke/1.0", "Set the User-Agent header")
	flag.StringVar(userAgent, "user-agent", "poke/1.0", "Set the User-Agent header")

	headers := flag.String("H", "", "Request headers (key:value)")
	flag.StringVar(headers, "headers", "", "Request headers (key:value)")

	verbose := flag.Bool("v", false, "Verbose output")
	flag.BoolVar(verbose, "verbose", false, "Verbose output")

	editor := flag.Bool("edit", false, "Open payload in editor")
	savePath := flag.String("save", "", "Save request to file")
	sendPath := flag.String("send", "", "Send request from file")
	flag.Parse()

	if *savePath != "" && *sendPath != "" {
		Error("Cannot use both -save and -send options at the same time", nil)
	}

	var req *SavedRequest

	if *sendPath != "" {
		// Load the request from the specified file
		filepath := resolveRequestPath(*sendPath)
		loaded, err := loadSavedRequest(filepath)
		if err != nil {
			Error("Failed to load request from file", err)
		}
		req = loaded

		// Overwrite the URL/headers/body if specified
		if len(flag.Args()) > 0 {
			req.URL = flag.Args()[0]
		}
		// don't overwrite the method if it is set to GET
		// since -X flag defaults to GET, it will always overwrite
		if flag.Lookup("X").Value.String() != "GET" {
			req.Method = *method
		}
		if *headers != "" {
			mergeHeaders(req.Headers, parseHeaders(*headers))
		}
		if *editor {
			// if -d is set, use that as the payload
			// otherwise, use the existing payload from the saved request
			if *data != "" {
				req.Body = resolvePayload(*data, *editor)
			} else {
				req.Body = resolvePayload(req.Body, *editor)
			}
		}
	} else {
		// Build request from flags
		if len(flag.Args()) < 1 {
			fmt.Println("Usage: poke [options] <url>")
			flag.PrintDefaults()
			os.Exit(1)
		}
		url := flag.Args()[0]
		headersMap := parseHeaders(*headers)
		body := resolvePayload(*data, *editor)

		req = &SavedRequest{
			Method:    *method,
			URL:       url,
			Headers:   headersMap,
			Body:      body,
			CreatedAt: time.Now(),
		}
	}

	if *userAgent != "" {
		req.Headers["User-Agent"] = *userAgent
	}

	if *savePath != "" {
		err := saveRequest(resolveRequestPath(*savePath), req, *data)
		if err != nil {
			Error("Failed to save request", err)
		}
		fmt.Printf("Request saved to %s\n", *savePath)
	}

	start := time.Now()
	resp, err := SendRequest(*req)
	duration := time.Since(start)

	if err != nil {
		Error("Request failed", err)
	}
	defer resp.Body.Close()

	bodyBytes := readResponse(resp)
	status := colorStatus(resp.StatusCode)

	fmt.Printf("[Status: %s]\n\n", status)

	if *verbose {
		fmt.Printf("URL: %s\n", req.URL)
		fmt.Printf("Method: %s\n", req.Method)
		fmt.Printf("Duration: %v\n", duration)
		fmt.Printf("Request Size: %d bytes\n", len(req.Body))
		fmt.Printf("Response Size: %d bytes\n", len(bodyBytes))
		fmt.Printf("Content-Type: %s\n\n", resp.Header.Get("Content-Type"))
	}

	printBody(bodyBytes, resp.Header.Get("Content-Type"))
}
