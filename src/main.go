package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	//create the ~/.poke/requests and collections directories if they don't exist
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error finding home directory:", err)
		return
	}
	requestsDir := fmt.Sprintf("%s/.poke/requests", home)
	collectionsDir := fmt.Sprintf("%s/.poke/collections", home)
	os.MkdirAll(requestsDir, 0755)
	os.MkdirAll(collectionsDir, 0755)

	flag.Usage = func() {
		fmt.Println("Usage: poke [command] [options] <args>")
		fmt.Println("Commands:")
		fmt.Println("  collections             List all collections")
		fmt.Println("  list <collection>       List all saved requests in a collection")
		fmt.Println("  send <file|collection>  Send request(s) from a file or collection")
		fmt.Println("  (default)               Send/Save a request")
		flag.PrintDefaults()
	}

	// Global flags.
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
	repeat := flag.Int("repeat", 1, "Number of times to send the request (across all workers)")
	workers := flag.Int("workers", 1, "Number of concurrent workers")
	expectStatus := flag.Int("expect-status", 0, "Expected status code")
	editor := flag.Bool("edit", false, "Open payload in editor")
	savePath := flag.String("save", "", "Save request to file (works with normal poke usage)")
	flag.Parse()

	args := flag.Args()
	// Subcommand dispatch.
	if len(args) > 0 {
		switch args[0] {
		case "collections":
			listCollections()
			return
		case "list":
			if len(args) < 2 {
				fmt.Println("Usage: poke list <collection>")
				os.Exit(1)
			}
			listCollection(args[1])
			return
		case "send":
			if len(args) < 2 {
				fmt.Println("Usage: poke send <file|collection>")
				os.Exit(1)
			}
			handleSendCommand(args[1], *verbose)
			return
		}
	}

	// Default behavior: build request from flags and send it.
	if len(args) < 1 {
		fmt.Println("Usage: poke [options] <url>")
		flag.PrintDefaults()
		os.Exit(1)
	}
	url := args[0]
	headersMap := parseHeaders(*headers)
	body := resolvePayload(*data, *editor)
	req := &PokeRequest{
		Method:       *method,
		URL:          url,
		Headers:      headersMap,
		Body:         body,
		CreatedAt:    time.Now(),
		Workers:      *workers,
		Repeat:       *repeat,
		ExpectStatus: *expectStatus,
	}

	if *userAgent != "" {
		req.Headers["User-Agent"] = *userAgent
	}

	if *savePath != "" {
		path := resolveRequestPath(*savePath)
		if err := saveRequest(path, req, *data); err != nil {
			Error("Failed to save request", err)
		}
		fmt.Printf("Request saved to %s\n", path)
	}

	if req.Repeat > 1 {
		if req.Workers > req.Repeat {
			req.Workers = req.Repeat
		}
		RunBenchmark(req, req.Repeat, req.Workers, req.ExpectStatus, *verbose)
		return
	} else {
		runRequest(req, *verbose)
	}
}
