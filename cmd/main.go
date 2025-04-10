package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"poke/core"
	"poke/types"
	"poke/util"
)

func main() {
	templater := &core.DefaultTemplateEngineImpl{}
	payloadResolver := &core.DefaultPayloadResolverImpl{}
	collectionHandler := &core.DefaultCollectionHandlerImpl{}
	requestRunner := &core.DefaultRequestRunnerImpl{}

	templater.LoadEnv()

	ensurePokeDirs()

	opts := parseCLIOptions()
	args := flag.Args()

	// Route CLI commands
	switch {
	case len(args) > 0 && args[0] == "collections":
		handleCollections(args, opts, collectionHandler)
		return
	case len(args) > 0 && args[0] == "send":
		handleSend(args, opts, collectionHandler)
		return
	case opts.Help:
		printUsage()
		return
	case len(args) < 1:
		fmt.Println("Usage: poke [options] <url>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Default single or benchmarked request
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
		Method:       opts.Method,
		URL:          url,
		Headers:      headers,
		QueryParams:  queryParams,
		Body:         payload,
		BodyFile:     opts.DataFile,
		BodyStdin:    false,
		CreatedAt:    time.Now(),
		Workers:      opts.Workers,
		Repeat:       opts.Repeat,
		ExpectStatus: opts.ExpectStatus,
	}

	if opts.UserAgent != "" {
		req.Headers["User-Agent"] = opts.UserAgent
	}

	if opts.SavePath != "" {
		savePath := util.ResolveRequestPath(opts.SavePath)
		if err := requestRunner.Save(req, savePath); err != nil {
			util.Error("Failed to save request", err)
		}
		fmt.Printf("Request saved to %s\n", savePath)
	}

	if err := requestRunner.Execute(req, opts.Verbose); err != nil {
		util.Error("Failed to execute request", err)
	}
}

func ensurePokeDirs() {
	home, err := os.UserHomeDir()
	if err != nil {
		util.Error("Could not determine home directory", err)
	}
	dirs := []string{
		filepath.Join(home, ".poke", "requests"),
		filepath.Join(home, ".poke", "collections"),
	}
	for _, dir := range dirs {
		_ = os.MkdirAll(dir, 0755)
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
	fmt.Println("  collections             List all collections")
	fmt.Println("  send <file|collection>  Send request(s) from a file or collection")
	flag.PrintDefaults()
}

func handleCollections(args []string, opts *types.CLIOptions, handler core.CollectionHandler) {
	if opts.Help {
		fmt.Println("Usage: poke collections [collection_name]")
	}
	if len(args) > 1 {
		_ = handler.List(args[1])
	} else {
		_ = handler.ListAll()
	}
}

func handleSend(args []string, opts *types.CLIOptions, handler core.CollectionHandler) {
	if opts.Help || len(args) < 2 {
		fmt.Println("Usage: poke send <file|collection>")
		os.Exit(1)
	}
	if err := handler.Send(args[1], opts.Verbose); err != nil {
		util.Error("Failed to send request(s)", err)
	}
}
