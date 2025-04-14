package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"poke/types"

	"github.com/TylerBrock/colorjson"
	"github.com/fatih/color"
)

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

	if len(resp.Headers) > 0 {
		fmt.Println("\nResponse Headers:")
		for k, v := range resp.Headers {
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

func AssertResponse(resp *types.PokeResponse, assertions *types.Assertions) (bool, error) {
	if assertions.Status != 0 && resp.StatusCode != assertions.Status {
		return false, fmt.Errorf("expected status %d, got %d", assertions.Status, resp.StatusCode)
	}

	if assertions.BodyContains != "" && !strings.Contains(string(resp.Body), assertions.BodyContains) {
		return false, fmt.Errorf("expected body to contain %q, got %q", assertions.BodyContains, string(resp.Body))
	}

	for k, v := range assertions.Headers {
		vals, ok := resp.Headers[k]
		if !ok {
			return false, fmt.Errorf("expected header %q to be %q, but it is missing", k, v)
		}
		if len(vals) == 0 {
			return false, fmt.Errorf("expected header %q to be %q, but it is empty", k, v)
		}
		if vals[0] != v {
			return false, fmt.Errorf("expected header %q to be %q, got %q", k, v, vals)
		}
	}

	return true, nil
}

func ParseHeaders(headerStr string) map[string][]string {
	headers := make(map[string][]string)
	if headerStr == "" {
		return headers
	}
	// Expect input like: "Key1:Value1;Key2:Value2"
	pairs := strings.Split(headerStr, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			headers[key] = append(headers[key], val)
		}
	}
	return headers
}

func ParseQueryParams(rawURL string) map[string][]string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return map[string][]string{}
	}
	return u.Query()
}

func MergeHeaders(base, extra map[string][]string) {
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

func ColorString(s string, colorName string) string {
	switch colorName {
	case "red":
		return color.New(color.FgRed).Sprintf("%s", s)
	case "green":
		return color.New(color.FgGreen).Sprintf("%s", s)
	case "yellow":
		return color.New(color.FgYellow).Sprintf("%s", s)
	case "blue":
		return color.New(color.FgBlue).Sprintf("%s", s)
	case "magenta":
		return color.New(color.FgMagenta).Sprintf("%s", s)
	case "cyan":
		return color.New(color.FgCyan).Sprintf("%s", s)
	default:
		return s
	}
}

func Error(msg string, err error) {
	if err == nil {
		fmt.Fprintf(os.Stderr, "[Error] %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "[Error] %s: %v\n", msg, err)
	}
	os.Exit(1)
}

func Debug(module string, msg string) {
	debug := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG")))
	if debug == "1" || debug == "true" {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", module, msg)
	}
}
