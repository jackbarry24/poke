package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"poke/core"
	"poke/types"
	"poke/util"
)

func main() {
	ensurePokeDir()
	opts := parseCLIOptions()
	args := flag.Args()

	runner := core.NewRequestRunner(opts)

	switch {
	case len(args) > 0 && args[0] == "send":
		handleSend(args, runner)
		return
	case opts.Help:
		printUsage()
		return
	case len(args) < 1:
		fmt.Println("Usage: poke [options] <url>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if opts.Workers > opts.Repeat {
		opts.Workers = opts.Repeat
	}

	rawURL := args[0]
	headers := util.ParseHeaders(opts.Headers)
	u, err := url.Parse(rawURL)
	if err != nil {
		util.Error("Failed to parse URL: %v", err)
	}

	payload, fromFile, err := runner.Pyld.Resolve(opts.Data, opts.DataFile, opts.DataStdin, opts.Editor)
	if err != nil {
		payload = opts.Data
		util.Warn("Failed to resolve payload...request body may not be as expected: %v", err)
	}

	req := &types.PokeRequest{
		Method:      opts.Method,
		FullURL:     rawURL,
		Scheme:      u.Scheme,
		Host:        u.Host,
		Path:        u.Path,
		Headers:     headers,
		QueryParams: u.Query(),
		Body:        payload,
		BodyFile:    opts.DataFile,
		Retries:     opts.Retries,
		Backoff:     opts.Backoff,
		Meta:        &types.Meta{CreatedAt: time.Now()},
		Workers:     opts.Workers,
		Repeat:      opts.Repeat,
		Assert:      &types.Assertions{Status: opts.ExpectStatus},
	}

	if opts.UserAgent != "" {
		req.Headers["User-Agent"] = []string{"poke/1.0"}
	}

	if len(req.Body) > 0 || req.BodyFile != "" || opts.DataStdin {
		req.Method = "POST"
	}

	req.ContentType = util.DetectContentType(req)
	if req.ContentType != "" {
		req.Headers["Content-Type"] = []string{req.ContentType}
	}

	if opts.SavePath != "" {
		if fromFile {
			req.Body = ""
		}
		if err := runner.SaveRequest(req, opts.SavePath); err != nil {
			util.Warn("Failed to save request: %v", err)
		}
		if opts.Verbose {
			util.Info("Request saved to: %s", opts.SavePath)
		}
	}

	if err := runner.Execute(req); err != nil {
		util.Error("Failed to execute request: %v", err)
	}
}

func parseCLIOptions() *types.CLIOptions {
	opts := &types.CLIOptions{}

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
	flag.IntVar(&opts.Retries, "retry", 1, "Retry request if response status is not 200 or does not match --expect-status")
	flag.IntVar(&opts.Backoff, "backoff", 1, "Base backoff duration in seconds")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "Render request but do not send")
	flag.BoolVar(&opts.Editor, "edit", false, "Open payload in editor")
	flag.StringVar(&opts.SavePath, "save", "", "Save request to file")
	flag.BoolVar(&opts.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&opts.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&opts.Help, "h", false, "Show help message")

	flag.Usage = printUsage
	flag.Parse()
	return opts
}

func printUsage() {
	fmt.Println("Usage: poke [command] [options] <args>")
	fmt.Println("Commands:")
	fmt.Println("  send  <path>  Send request(s) from a file/directory")
	flag.PrintDefaults()
}

func handleSend(args []string, runner *core.RequestRunnerImpl) {
	if runner.Opts.Help || len(args) < 2 {
		fmt.Println("Usage: poke send <path>")
		os.Exit(1)
	}

	if err := runner.Collect(args[1]); err != nil {
		util.Warn("Failed to send request(s): %v", err)
	}
}

func ensurePokeDir() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		util.Error("Failed to get home directory: %v", err)
	}
	pokeDir := fmt.Sprintf("%s/.poke", homedir)
	if _, err := os.Stat(pokeDir); os.IsNotExist(err) {
		err := os.MkdirAll(pokeDir, os.ModePerm)
		if err != nil {
			util.Error("Failed to create .poke directory: %v", err)
		}
	}

	tmpFilePath := fmt.Sprintf("%s/tmp_poke_latest.json", pokeDir)
	if _, err := os.Stat(tmpFilePath); os.IsNotExist(err) {
		file, err := os.Create(tmpFilePath)
		if err != nil {
			util.Error("Failed to create tmp_poke_latest.json file: %v", err)
		}
		defer file.Close()
	}
}
