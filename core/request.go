package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"poke/types"
	"poke/util"
)

type RequestRunner interface {
	Execute(req *types.PokeRequest) error
	Send(req *types.PokeRequest) (*types.PokeResponse, error)
	SendAndVerify(req *types.PokeRequest) (*types.PokeResponse, bool, error)
	Collect(path string) error
	SaveRequest(req *types.PokeRequest, saveAs string) error
	SaveResponse(resp *types.PokeResponse) error
	Load(path string) (*types.PokeRequest, error)
}

type RequestRunnerImpl struct {
	Tmpl *TemplateEngineImpl
	Pyld *PayloadResolverImpl
	Opts *types.CLIOptions
}

func NewRequestRunner(opts *types.CLIOptions) *RequestRunnerImpl {
	return &RequestRunnerImpl{
		Tmpl: &TemplateEngineImpl{},
		Pyld: &PayloadResolverImpl{},
		Opts: opts,
	}
}

// Execute sends one or more requests, handling dry-run, retries, concurrency, and output.
// For a single request, prints the response; for multiple, prints a benchmark summary.
func (r *RequestRunnerImpl) Execute(req *types.PokeRequest) error {
	if r.Opts.DryRun {
		util.DumpRequest(req)
		return nil
	}

	if req.Workers < 1 {
		req.Workers = 1
	}
	if req.Workers > req.Repeat {
		req.Workers = req.Repeat
	}

	results, totalTime := r.dispatch(req)

	if req.Repeat <= 1 {
		if len(results) == 1 {
			res := results[0]
			if res.Err != nil || !res.Ok {
				util.Warn("Request failed: %v", res.Err)
			}
			_ = r.SaveResponse(res.Resp)
			if res.Ok {
				if r.Opts.Verbose {
					util.PrintResponseVerbose(res.Resp, req, res.Resp.Duration)
				} else {
					if res.Resp.StatusCode != 404 {
						util.PrintBody(res.Resp.Body, res.Resp.ContentType)
					} else {
						util.Info("%s", util.ColorStatus(res.Resp.StatusCode))
					}
				}
			}
		}
		return nil
	}

	var successes, failures int
	var durations []time.Duration
	for _, res := range results {
		if res.Err != nil || !res.Ok {
			failures++
		} else {
			successes++
			durations = append(durations, res.Resp.Duration)
		}
	}
	bench := types.BenchmarkResult{
		Total:     req.Repeat,
		Successes: successes,
		Failures:  failures,
		Durations: durations,
	}
	util.PrintBenchmarkResults(bench, totalTime.Seconds(), req)
	return nil
}

// execResult holds the outcome of a single HTTP request send & verify.
type execResult struct {
	Resp *types.PokeResponse
	Ok   bool
	Err  error
}

// sendWithRetries performs up to req.Retries attempts, with exponential backoff on failure.
func (r *RequestRunnerImpl) sendWithRetries(req *types.PokeRequest) (*types.PokeResponse, bool, error) {
	base := time.Duration(req.Backoff) * time.Second
	max := base + 10*time.Second
	var resp *types.PokeResponse
	var err error
	ok := false
	for i := range req.Retries {
		resp, ok, err = r.SendAndVerify(req)
		if ok {
			break
		}
		backoff := util.Backoff(base, max, i)
		if r.Opts.Verbose && req.Retries > 1 {
			util.Info("Attempt %d failed", i+1)
			util.Info("Retrying... backoff %.3fs", backoff.Seconds())
		}
		if i < req.Retries-1 {
			time.Sleep(backoff)
		}
	}
	return resp, ok, err
}

// dispatch runs req.Repeat request jobs across up to req.Workers goroutines.
// Returns a slice of execResult and the total elapsed time.
func (r *RequestRunnerImpl) dispatch(req *types.PokeRequest) ([]execResult, time.Duration) {
	count := req.Repeat
	jobs := make(chan int, count)
	for i := 1; i <= count; i++ {
		jobs <- i
	}
	close(jobs)
	results := make(chan execResult, count)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < req.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for num := range jobs {
				resp, ok, err := r.sendWithRetries(req)
				if r.Opts.Verbose && resp != nil {
					util.Info("Req %-3d: %s (%v)", num, util.ColorStatus(resp.StatusCode), resp.Duration)
				}
				results <- execResult{Resp: resp, Ok: ok, Err: err}
			}
		}()
	}
	wg.Wait()
	total := time.Since(start)
	close(results)
	out := make([]execResult, 0, count)
	for res := range results {
		out = append(out, res)
	}
	return out, total
}

// SendAndVerify sends the HTTP request and applies assertions.
func (r *RequestRunnerImpl) SendAndVerify(req *types.PokeRequest) (*types.PokeResponse, bool, error) {
	resp, err := r.Send(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Raw.Body.Close()
	if req.Assert != nil {
		ok, err := util.AssertResponse(resp, req.Assert)
		if !ok {
			return resp, false, err
		}
	}
	return resp, true, nil
}

// Send constructs and sends the HTTP request, capturing timing and response.
func (r *RequestRunnerImpl) Send(req *types.PokeRequest) (*types.PokeResponse, error) {
	client := &http.Client{}
	httpReq, err := http.NewRequest(req.Method, req.FullURL, bytes.NewBufferString(req.Body))
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, strings.Join(v, ","))
	}
	start := time.Now()
	rawResp, err := client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := util.ReadResponse(rawResp)
	if err != nil {
		return nil, err
	}
	return &types.PokeResponse{
		StatusCode:  rawResp.StatusCode,
		Headers:     rawResp.Header,
		Body:        bodyBytes,
		ContentType: rawResp.Header.Get("Content-Type"),
		Raw:         rawResp,
		Timestamp:   time.Now(),
		Duration:    duration,
	}, nil
}

// SaveRequest writes a PokeRequest to path, clearing Body if BodyFile is set.
func (r *RequestRunnerImpl) SaveRequest(req *types.PokeRequest, path string) error {
	if req.BodyFile != "" {
		req.Body = ""
	}
	out, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return os.WriteFile(path, out, 0644)
}

// SaveResponse writes the latest response to ~/.poke/tmp_poke_latest.json
func (r *RequestRunnerImpl) SaveResponse(resp *types.PokeResponse) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}
	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(home, ".poke", "tmp_poke_latest.json"), out, 0644)
}

// Collect loads and sends saved request(s) from a file or directory.
// Collect loads and sends one or more saved requests from a file or directory.
func (r *RequestRunnerImpl) Collect(path string) error {
	paths, err := walkPath(path)
	if err != nil {
		return fmt.Errorf("could not resolve file/directory: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no .json files found for '%s'", path)
	}
	for i, p := range paths {
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Request %d/%d: %s\n", i+1, len(paths), p)
		fmt.Println(strings.Repeat("-", 40))
		req, err := r.Load(p)
		if err != nil {
			fmt.Printf("File '%s' is not a valid request: %v\n", p, err)
			continue
		}
		body, err := r.Pyld.Resolve(string(req.Body), req.BodyFile, false, false)
		if err != nil {
			fmt.Printf("Failed to resolve body for '%s': %v\n", p, err)
			continue
		}
		req.Body = body
		if err := r.Execute(req); err != nil {
			fmt.Printf("Request failed: %v\n", err)
		}
	}
	return nil
}

// Load reads and renders a .json request template into a PokeRequest.
func (r *RequestRunnerImpl) Load(fpath string) (*types.PokeRequest, error) {
	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	req, err := r.Tmpl.RenderRequest(data)
	if err != nil {
		return nil, err
	}
	if req.BodyFile != "" {
		content, err := os.ReadFile(req.BodyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read body file: %w", err)
		}
		req.Body = string(content)
	}
	scheme := req.Scheme
	if scheme == "" {
		scheme = "http"
	}
	host := req.Host
	path := req.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var queryStr string
	if len(req.QueryParams) > 0 {
		parts := []string{}
		for k, vs := range req.QueryParams {
			for _, v := range vs {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
		}
		queryStr = "?" + strings.Join(parts, "&")
	}
	req.FullURL = fmt.Sprintf("%s://%s%s%s", scheme, host, path, queryStr)
	req.ContentType = util.DetectContentType(req)
	if req.ContentType != "" {
		req.Headers["Content-Type"] = []string{req.ContentType}
	}
	return req, nil
}

// walkPath collects .json files from a path (file or directory).
func walkPath(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		return nil, fmt.Errorf("could not access path: %s", path)
	}
	var paths []string
	if info.IsDir() {
		filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() && strings.HasSuffix(fi.Name(), ".json") {
				paths = append(paths, p)
			}
			return nil
		})
	} else {
		paths = append(paths, path)
	}
	return paths, nil
}
