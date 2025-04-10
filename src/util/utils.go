package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"poke/types"

	"github.com/TylerBrock/colorjson"
	"github.com/fatih/color"
)

// ========== Request I/O ==========

func ResolveRequestPath(input string) string {
	if strings.Contains(input, "/") {
		return input
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return input
	}

	dir := filepath.Join(home, ".poke")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, input)
}

// ========== Response Output ==========

func ReadResponse(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
}

func PrintResponseVerbose(resp *types.PokeResponse, req *types.PokeRequest, body []byte, duration float64) {
	status := ColorStatus(resp.StatusCode)

	fmt.Println("──────────────────────Response Data──────────────────────")
	fmt.Printf("Status:             %s\n", status)
	fmt.Printf("URL:                %s\n", req.URL)
	fmt.Printf("Method:             %s\n", req.Method)
	fmt.Printf("Duration:           %.2fs\n", duration)
	fmt.Printf("Request Size:       %d bytes\n", len(req.Body))
	fmt.Printf("Response Size:      %d bytes\n", len(body))
	fmt.Printf("Content-Type:       %s\n", resp.ContentType)

	if len(resp.Header) > 0 {
		fmt.Println("\nResponse Headers:")
		for k, v := range resp.Header {
			fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
		}
	}

	fmt.Println("──────────────────────Response Body──────────────────────")
	PrintBody(body, resp.ContentType)
}

func PrintBody(body []byte, contentType string) {
	if strings.Contains(contentType, "application/json") {
		var obj interface{}
		err := json.Unmarshal(body, &obj)
		if err != nil {
			fmt.Println(string(body)) // fallback to raw
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

// ========== CLI Helpers ==========

func ParseHeaders(headerStr string) map[string]string {
	headers := make(map[string]string)
	if headerStr == "" {
		return headers
	}
	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			headers[key] = val
		}
	}
	return headers
}

func MergeHeaders(base, extra map[string]string) {
	for k, v := range extra {
		base[k] = v
	}
}

func ColorStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return color.New(color.FgGreen).Sprintf("%d OK", code)
	case code >= 300 && code < 400:
		return color.New(color.FgYellow).Sprintf("%d Redirect", code)
	case code >= 400:
		return color.New(color.FgRed).Sprintf("%d Error", code)
	default:
		return fmt.Sprintf("%d", code)
	}
}

// ========== Error Exit ==========

func Error(msg string, err error) {
	if err == nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	}
	os.Exit(1)
}
