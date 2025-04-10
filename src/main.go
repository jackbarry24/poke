package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func ParseCLIOptions() *CLIOptions {
	opts := &CLIOptions{}

	flag.StringVar(&opts.Method, "X", "GET", "HTTP method to use")
	flag.StringVar(&opts.Method, "method", "GET", "HTTP method to use")

	flag.StringVar(&opts.Data, "d", "", "Request body payload")
	flag.StringVar(&opts.Data, "data", "", "Request body payload")
	flag.StringVar(&opts.DataFile, "data-file", "", "File containing request body payload")
	flag.BoolVar(&opts.DataStdin, "data-stdin", false, "Read request body from stdin")

	flag.StringVar(&opts.UserAgent, "A", "poke/1.0", "Set the User-Agent header")
	flag.StringVar(&opts.UserAgent, "user-agent", "poke/1.0", "Set the User-Agent header")
	flag.StringVar(&opts.Headers, "H", "", "Request headers (key:value)")
	flag.StringVar(&opts.Headers, "headers", "", "Request headers (key:value)")

	flag.IntVar(&opts.Repeat, "repeat", 1, "Number of times to send the request (across all workers)")
	flag.IntVar(&opts.Workers, "workers", 1, "Number of concurrent workers")
	flag.IntVar(&opts.ExpectStatus, "expect-status", 0, "Expected status code")
	flag.BoolVar(&opts.Editor, "edit", false, "Open payload in editor")
	flag.StringVar(&opts.SavePath, "save", "", "Save request to file")

	flag.BoolVar(&opts.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&opts.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&opts.Help, "h", false, "Show help message")

	flag.Usage = func() {
		fmt.Println("Usage: poke [command] [options] <args>")
		fmt.Println("Commands:")
		fmt.Println("  collections             List all collections")
		fmt.Println("  send <file|collection>  Send request(s) from a file or collection")
		fmt.Println("  (default)               Send/Save a request")
		flag.PrintDefaults()
	}

	flag.Parse()
	return opts
}

func main() {

	loadEnvFile()

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error finding home directory:", err)
		return
	}
	os.MkdirAll(fmt.Sprintf("%s/.poke/requests", home), 0755)
	os.MkdirAll(fmt.Sprintf("%s/.poke/collections", home), 0755)

	opts := ParseCLIOptions()
	args := flag.Args()

	if len(args) > 0 {
		switch args[0] {
		case "collections":
			if opts.Help {
				fmt.Println("Usage: poke collections [collection_name]")
			}
			if len(args) > 1 {
				ListCollection(args[1])
			} else {
				ListCollections()
			}
			return
		case "send":
			if opts.Help || len(args) < 2 {
				fmt.Println("Usage: poke send <file|collection>")
				os.Exit(1)
			}
			HandleSendCommand(args[1], opts)
			return
		}
	}

	if opts.Help {
		flag.Usage()
		return
	}

	if len(args) < 1 {
		fmt.Println("Usage: poke [options] <url>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if opts.Workers > opts.Repeat {
		opts.Workers = opts.Repeat
	}

	url := args[0]
	headersMap := parseHeaders(opts.Headers)
	body := resolvePayload(opts.Data, opts.DataFile, opts.DataStdin, opts.Editor)
	req := &PokeRequest{
		Method:       opts.Method,
		URL:          url,
		Headers:      headersMap,
		Body:         body,
		BodyFile:     opts.DataFile,
		BodyStdin:    opts.DataStdin,
		CreatedAt:    time.Now(),
		Workers:      opts.Workers,
		Repeat:       opts.Repeat,
		ExpectStatus: opts.ExpectStatus,
	}

	if opts.UserAgent != "" {
		req.Headers["User-Agent"] = opts.UserAgent
	}

	if opts.SavePath != "" {
		path := resolveRequestPath(opts.SavePath)
		if err := saveRequest(path, req); err != nil {
			Error("Failed to save request", err)
		}
		fmt.Printf("Request saved to %s\n", path)
	}

	RunRequest(req, opts.Verbose)
}
