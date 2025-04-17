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

	for i := 0; i < req.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workload := base
			if workerID < remainder {
				workload++
			}
			for j := 0; j < workload; j++ {
				t0 := time.Now()
				resp, err := r.Send(req)
				duration := time.Since(t0)

				reqNum := atomic.AddInt64(&counter, 1)
				if r.Opts.Verbose {
					status := "ERR"
					if err == nil {
						status = util.ColorStatus(resp.StatusCode)
					}
					fmt.Printf("Request %-3d [Worker %-2d]: %-7s (%v)\n", reqNum, workerID, status, duration)
				}

				if err != nil {
					errorChan <- true
					continue
				}
				resp.Raw.Body.Close()

				if req.Assert != nil {
					ok, _ := util.AssertResponse(resp, req.Assert)
					if !ok {
						errorChan <- true
						continue
					}
				}

				errorChan <- false
				resultChan <- duration
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
	var duration float64
	var resp *types.PokeResponse
	var err error
	success := false
	i := 0

	for !success && i < req.Retries {
		util.Debug("runner", fmt.Sprintf("attempt: %d url: %s", i+1, req.URL))
		start := time.Now()
		resp, err = r.Send(req)
		duration = time.Since(start).Seconds()

		if err != nil {
			if i < req.Retries-1 {
				fmt.Printf("attempt %d failed: %v, ...retrying\n", i+1, err)
				time.Sleep(time.Second)
			}
			i += 1
			continue
		}

		if err = r.SaveResponse(resp); err != nil {
			util.Debug("runner", "failed to save latest response")
		}

		if err != nil {
			if i < req.Retries-1 {
				fmt.Printf("attempt %d failed: %v, ...retrying\n", i+1, err)
				time.Sleep(time.Second)
			}
			i += 1
			continue
		}
		defer resp.Raw.Body.Close()

		if req.Assert != nil {
			var ok bool
			ok, err = util.AssertResponse(resp, req.Assert)
			if !ok {
				if i < req.Retries-1 {
					fmt.Printf("attempt %d failed: %v, ...retrying\n", i+1, err)
					time.Sleep(time.Second)
				}
				i += 1
				continue
			}
		}
		success = true
	}

	if !success {
		util.Error(fmt.Sprintf("Request failed after %d attempt(s): %v\n", req.Retries, err), nil)
	}

	if r.Opts.Verbose {
		util.PrintResponseVerbose(resp, req, resp.Body, duration)
	} else {
		fmt.Printf("%s\n\n", util.ColorStatus(resp.StatusCode))
		util.PrintBody(resp.Body, resp.ContentType)
	}
	return nil
}

func (r *RequestRunnerImpl) Send(req *types.PokeRequest) (*types.PokeResponse, error) {
	client := &http.Client{}
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBufferString(req.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, strings.Join(v, ","))
	}

	resp, err := client.Do(httpReq)
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
	}, nil
}

func (r *RequestRunnerImpl) SaveRequest(req *types.PokeRequest, path string) error {
	if req.BodyFile != "" {
		req.Body = ""
	}
	if req.BodyStdin {
		req.BodyStdin = false
	}

	if strings.Contains(req.URL, "?") {
		parts := strings.Split(req.URL, "?")
		req.URL = parts[0]
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
			body, err := r.Pyld.Resolve(req.Body, req.BodyFile, req.BodyStdin, false)
			if err != nil {
				return fmt.Errorf("failed to resolve payload: %w", err)
			}
			req.Body = body
			req.BodyFile = ""
			req.BodyStdin = false
			return r.Execute(req)
		}
	}

	paths, err := walkPath(path)
	if err != nil {
		return fmt.Errorf("could not resolve collection: %w", err)
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

		body, err := r.Pyld.Resolve(req.Body, req.BodyFile, req.BodyStdin, false)
		if err != nil {
			fmt.Printf("Failed to resolve body for '%s': %v\n", path, err)
			continue
		}
		req.Body = body
		req.BodyFile = ""
		req.BodyStdin = false

		err = r.Execute(req)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
		}
		fmt.Println()
	}
	return nil
}

func (r *RequestRunnerImpl) Load(path string) (*types.PokeRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	req, err := r.Tmpl.RenderRequest(data)
	if err != nil {
		return nil, err
	}

	if len(req.QueryParams) > 0 && !strings.Contains(req.URL, "?") {
		query := "?"
		for k, vals := range req.QueryParams {
			for _, v := range vals {
				query += fmt.Sprintf("%s=%s&", k, v)
			}
		}
		query = strings.TrimSuffix(query, "&")
		req.URL += query
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
