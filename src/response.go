package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func readResponse(resp *http.Response) []byte {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return bodyBytes
}

func printResponseVerbose(resp *http.Response, req *PokeRequest, body []byte, duration time.Duration) {
	status := colorStatus(resp.StatusCode)

	fmt.Println("──────────────────────Response Data──────────────────────")
	fmt.Printf("Status:             %s\n", status)
	fmt.Printf("URL:                %s\n", req.URL)
	fmt.Printf("Method:             %s\n", req.Method)
	fmt.Printf("Duration:           %v\n", duration)
	fmt.Printf("Request Size:       %d bytes\n", len(req.Body))
	fmt.Printf("Response Size:      %d bytes\n", len(body))
	fmt.Printf("Content-Type:       %s\n", resp.Header.Get("Content-Type"))

	if len(resp.Header) > 0 {
		fmt.Println("\nResponse Headers:")
		for k, v := range resp.Header {
			fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
		}
	}

	fmt.Println("──────────────────────Response Body──────────────────────")

	printBody(body, resp.Header.Get("Content-Type"))
}
