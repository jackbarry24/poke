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
	"sync/atomic"
	"time"

	"poke/types"
	"poke/util"
)

type RequestRunner interface {
	Execute(req *types.PokeRequest) error
	Send(req *types.PokeRequest) (*types.PokeResponse, error)
	SendAndVerify(req *types.PokeRequest) (*types.PokeResponse, error)
	RunBenchmark(req *types.PokeRequest) error
	RunSingleRequest(req *types.PokeRequest) error
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

func (r *RequestRunnerImpl) Execute(req *types.PokeRequest) error {
	if r.Opts.DryRun {
		util.DumpRequest(req)
		return nil
	}

	if req.Repeat > 1 {
		return r.RunBenchmark(req)
	}
	return r.RunSingleRequest(req)
}

func (r *RequestRunnerImpl) RunBenchmark(req *types.PokeRequest) error {
	var wg sync.WaitGroup
	resultChan := make(chan time.Duration, req.Repeat)
	errorChan := make(chan bool, req.Repeat)

	startTime := time.Now()

	base := req.Repeat / req.Workers
	remainder := req.Repeat % req.Workers
	var counter int64

	for i := range req.Workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workload := base
			if workerID < remainder {
				workload++
			}
			for range workload {
				resp, ok, err := r.SendAndVerify(req)

				reqNum := atomic.AddInt64(&counter, 1)
				if r.Opts.Verbose && resp != nil {
					status := util.ColorStatus(resp.StatusCode)
					util.Info("Req %-3d Worker %-2d: %-7s (%v)", reqNum, workerID, status, resp.Duration)
				}

				if err != nil || !ok {
					errorChan <- true
					continue
				}

				errorChan <- false
				if resp != nil {
					resultChan <- resp.Duration
				}
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(startTime)
	close(resultChan)
	close(errorChan)

	var durations []time.Duration
	var successes, failures int

	for err := range errorChan {
		if err {
			failures++
		} else {
			successes++
		}
	}

	for d := range resultChan {
		durations = append(durations, d)
	}

	result := types.BenchmarkResult{
		Total:     req.Repeat,
		Successes: successes,
		Failures:  failures,
		Durations: durations,
	}

	util.PrintBenchmarkResults(result, totalTime.Seconds(), req)
	return nil
}

func (r *RequestRunnerImpl) RunSingleRequest(req *types.PokeRequest) error {
	var resp *types.PokeResponse
	var err error
	ok := false

	baseBackoff := time.Duration(req.Backoff) * time.Second
	maxBackoff := baseBackoff + (10 * time.Second)

	for i := range req.Retries {
		resp, ok, err = r.SendAndVerify(req)
		if ok {
			break
		}
		backoff := util.Backoff(baseBackoff, maxBackoff, i)
		if r.Opts.Verbose && req.Retries > 1 {
			util.Info("Attempt %d failed", i+1)
		}
		if i < req.Retries-1 {
			if r.Opts.Verbose {
				util.Info("Retrying...backing off for %.3f seconds", backoff.Seconds())
			}
			time.Sleep(backoff)
		}

	}

	if err != nil || !ok {
		util.Warn("Request failed afer %d attempt(s): %v", req.Retries, err)
	}

	if err := r.SaveResponse(resp); err != nil {
		util.Warn("Failed to save latest response...history may not work as expected: %v", err)
	}

	if ok {
		if r.Opts.Verbose {
			util.PrintResponseVerbose(resp, req, resp.Duration)
		} else {
			if resp.StatusCode != 404 {
				util.PrintBody(resp.Body, resp.ContentType)
			} else {
				util.Info("%s", util.ColorStatus(resp.StatusCode))
			}
		}
	}
	return nil
}

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
	resp, err := client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := util.ReadResponse(resp)
	if err != nil {
		return nil, err
	}

	return &types.PokeResponse{
		StatusCode:  resp.StatusCode,
		Headers:     resp.Header,
		Body:        bodyBytes,
		ContentType: resp.Header.Get("Content-Type"),
		Raw:         resp,
		Timestamp:   time.Now(),
		Duration:    duration,
	}, nil
}

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

func (r *RequestRunnerImpl) SaveResponse(resp *types.PokeResponse) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(homeDir, ".poke", "tmp_poke_latest.json"), out, 0644)
}

func (r *RequestRunnerImpl) Collect(path string) error {
	if strings.HasSuffix(path, ".json") {
		if _, err := os.Stat(path); err == nil {
			req, err := r.Load(path)
			if err != nil {
				return fmt.Errorf("failed to load request: %w", err)
			}
			body, err := r.Pyld.Resolve(string(req.Body), req.BodyFile, false, false)
			if err != nil {
				return fmt.Errorf("failed to resolve payload: %w", err)
			}
			req.Body = body
			return r.Execute(req)
		}
	}

	paths, err := walkPath(path)
	if err != nil {
		return fmt.Errorf("could not resolve file/directory: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no .json files found for '%s'", path)
	}

	for i, path := range paths {
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Request %d/%d: %s\n", i+1, len(paths), path)
		fmt.Println(strings.Repeat("-", 40))

		req, err := r.Load(path)
		if err != nil {
			fmt.Printf("File '%s' is not a valid request: %v\n", path, err)
			continue
		}

		body, err := r.Pyld.Resolve(string(req.Body), req.BodyFile, false, false)
		if err != nil {
			fmt.Printf("Failed to resolve body for '%s': %v\n", path, err)
			continue
		}
		req.Body = body

		err = r.Execute(req)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
		}
	}
	return nil
}

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
		q := make([]string, 0)
		for k, vals := range req.QueryParams {
			for _, v := range vals {
				q = append(q, fmt.Sprintf("%s=%s", k, v))
			}
		}
		queryStr = "?" + strings.Join(q, "&")
	}
	req.FullURL = fmt.Sprintf("%s://%s%s%s", scheme, host, path, queryStr)

	req.ContentType = util.DetectContentType(req)
	if req.ContentType != "" {
		req.Headers["Content-Type"] = []string{req.ContentType}
	}

	return req, nil
}

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
		err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !fi.IsDir() && strings.HasSuffix(fi.Name(), ".json") {
				paths = append(paths, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		paths = append(paths, path)
	}
	return paths, nil
}
