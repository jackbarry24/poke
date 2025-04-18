package util

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"poke/types"

	"github.com/TylerBrock/colorjson"
	"github.com/fatih/color"
)

func ReadResponse(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
}

func PrintResponseVerbose(resp *types.PokeResponse, req *types.PokeRequest, body []byte, duration time.Duration) {
	status := ColorStatus(resp.StatusCode)

	fmt.Printf("-> %s %s\n", req.Method, req.Path)
	fmt.Printf("-> Host: %s\n", req.Host)
	for k, v := range req.Headers {
		fmt.Printf("-> %s: %s\n", k, strings.Join(v, ", "))
	}

	fmt.Println()
	fmt.Printf("<- %s\n", status)
	for k, v := range resp.Headers {
		fmt.Printf("<- %s: %s\n", k, strings.Join(v, ", "))
	}
	fmt.Println("<-")
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

func PrintBenchmarkResults(res types.BenchmarkResult, totalTime float64, req *types.PokeRequest) {
	fmt.Println("╭──────────── Poke Benchmark ────────────╮")
	fmt.Printf("│ Requests       %-23d │\n", res.Total)
	fmt.Printf("│ Success        %-32s │\n", ColorString(fmt.Sprintf("%d", res.Successes), "green"))
	fmt.Printf("│ Failures       %-32s │\n", ColorString(fmt.Sprintf("%d", res.Failures), "red"))
	fmt.Printf("│ Total time     %-.2fs%18s │\n", totalTime, "")

	if len(res.Durations) == 0 {
		fmt.Printf("│ Avg duration   %-23s │\n", "N/A")
		fmt.Printf("│ Min            %-23s │\n", "N/A")
		fmt.Printf("│ Max            %-23s │\n", "N/A")
	} else {
		min, max := res.Durations[0], res.Durations[0]
		var sum time.Duration
		for _, d := range res.Durations {
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
			sum += d
		}
		avg := sum / time.Duration(len(res.Durations))
		fmt.Printf("│ Avg duration   %-23v │\n", avg)
		fmt.Printf("│ Min            %-32s │\n", ColorString(min.String(), "blue"))
		fmt.Printf("│ Max            %-32s │\n", ColorString(max.String(), "yellow"))
	}

	throughput := float64(res.Total) / totalTime
	fmt.Printf("│ Throughput     %-23.2f │\n", throughput)
	fmt.Printf("│ Workers        %-23d │\n", req.Workers)
	fmt.Println("╰────────────────────────────────────────╯")
}

func AssertResponse(resp *types.PokeResponse, assertions *types.Assertions) (bool, error) {
	if assertions.Status != 0 && resp.StatusCode != assertions.Status {
		return false, fmt.Errorf("expected status %d, got %d", assertions.Status, resp.StatusCode)
	}

	if assertions.BodyContains != "" && !strings.Contains(string(resp.Body), assertions.BodyContains) {
		return false, fmt.Errorf("expected body to contain %q, got %q", assertions.BodyContains, string(resp.Body))
	}

	for k, expectedVals := range assertions.Headers {
		actualVals, ok := resp.Headers[k]
		if !ok {
			return false, fmt.Errorf("expected header %q to be %q, but it is missing", k, strings.Join(expectedVals, ", "))
		}
		if len(actualVals) == 0 {
			return false, fmt.Errorf("expected header %q to be %q, but it is empty", k, strings.Join(expectedVals, ", "))
		}
		for _, expectedVal := range expectedVals {
			found := false
			for _, actualVal := range actualVals {
				if actualVal == expectedVal {
					found = true
					break
				}
			}
			if !found {
				return false, fmt.Errorf("expected header %q to contain %q, but it was not found", k, expectedVal)
			}
		}
	}

	return true, nil
}

func ParseHeaders(headerStr string) map[string][]string {
	headers := make(map[string][]string)
	if headerStr == "" {
		return headers
	}

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

func Error(msg string, err error, exit bool) {
	if err == nil {
		fmt.Fprintf(os.Stderr, "[Error] %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "[Error] %s: %v\n", msg, err)
	}
	if exit {
		os.Exit(1)
	}
}

func Debug(module string, format string, args ...interface{}) {
	debug := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG")))
	if debug == "1" || debug == "true" {
		fmt.Fprintf(os.Stderr, "[%s] ", module)
		fmt.Fprintf(os.Stderr, format+"\n", args)
	}
}

func Info(format string, args ...interface{}) {
	fmt.Printf("[Info] "+format+"\n", args...)
}

func DumpRequest(req *types.PokeRequest) {
	if req == nil {
		return
	}

	fmt.Printf("Method: %s\n", req.Method)
	fmt.Printf("Host: %s\n", req.Host)
	if len(req.Headers) > 0 {
		fmt.Printf("Headers:\n")
		for k, v := range req.Headers {
			fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
		}
	}
	if len(req.Body) > 0 {
		fmt.Printf("Data: %s", req.Body)
	}
	if len(req.BodyFile) > 0 {
		fmt.Printf("Data File: %s", req.BodyFile)
	}
}

func Backoff(base, max time.Duration, attempt int) time.Duration {
	backoff := base * (1 << attempt)
	if backoff > max {
		backoff = max
	}

	jitter := time.Duration(rand.Int63n(int64(backoff)))
	return backoff + jitter
}
