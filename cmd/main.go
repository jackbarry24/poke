package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"poke/core"
	"poke/types"
	"poke/util"
)

func main() {
	payloadResolver := &core.DefaultPayloadResolverImpl{}
	requestRunner := &core.DefaultRequestRunnerImpl{}

	ensurePokeDir()
	opts := parseCLIOptions()
	args := flag.Args()

	switch {
	case len(args) > 0 && args[0] == "send":
		handleSend(args, opts, requestRunner)
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

	url := args[0]
	headers := util.ParseHeaders(opts.Headers)
	queryParams := util.ParseQueryParams(url)
	payload, err := payloadResolver.Resolve(opts.Data, opts.DataFile, opts.DataStdin, opts.Editor)
	if err != nil {
		util.Error("Failed to resolve payload", err)
	}

	req := &types.PokeRequest{
		Method:      opts.Method,
		URL:         url,
		Headers:     headers,
		QueryParams: queryParams,
		Body:        payload,
		BodyFile:    opts.DataFile,
		BodyStdin:   false,
		Meta:        &types.Meta{CreatedAt: time.Now()},
		Workers:     opts.Workers,
		Repeat:      opts.Repeat,
		Assert:      &types.Assertions{Status: opts.ExpectStatus},
	}

	if opts.UserAgent != "" {
		req.Headers["User-Agent"] = []string{"poke/1.0"}
	}

	if req.Body != "" || req.BodyFile != "" || req.BodyStdin {
		req.Method = "POST"
	}

	if opts.SavePath != "" {
		if err := requestRunner.SaveRequest(req, opts.SavePath); err != nil {
			util.Error("Failed to save request", err)
		}
		fmt.Printf("Request saved to %s\n", opts.SavePath)
	}

	if err := requestRunner.Execute(req, opts.Verbose); err != nil {
		util.Error("Failed to execute request", err)
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

func handleSend(args []string, opts *types.CLIOptions, handler core.RequestRunner) {
	if opts.Help || len(args) < 2 {
		fmt.Println("Usage: poke send <path>")
		os.Exit(1)
	}

	if err := handler.Collect(args[1], opts.Verbose); err != nil {
		util.Error("Failed to send request(s)", err)
	}
}

func ensurePokeDir() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		util.Error("Failed to get home directory", err)
	}
	pokeDir := fmt.Sprintf("%s/.poke", homedir)
	if _, err := os.Stat(pokeDir); os.IsNotExist(err) {
		err := os.MkdirAll(pokeDir, os.ModePerm)
		if err != nil {
			util.Error("Failed to create .poke directory", err)
		}
	}

	tmpFilePath := fmt.Sprintf("%s/tmp_poke_latest.json", pokeDir)
	if _, err := os.Stat(tmpFilePath); os.IsNotExist(err) {
		file, err := os.Create(tmpFilePath)
		if err != nil {
			util.Error("Failed to create tmp_poke_latest.json file", err)
		}
		defer file.Close()
	}
}
