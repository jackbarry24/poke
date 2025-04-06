package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TylerBrock/colorjson"
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

func printBody(body []byte, contentType string) {
	if strings.Contains(contentType, "application/json") {
		var obj interface{}
		err := json.Unmarshal(body, &obj)
		if err != nil {
			fmt.Println(string(body)) // fallback raw
			return
		}
		f := colorjson.NewFormatter()
		f.Indent = 2
		s, _ := f.Marshal(obj)
		fmt.Println(string(s))
	} else {
		fmt.Println(string(body))
	}
}
