package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"poke/types"
	"poke/util"
)

type RequestRunner interface {
	Execute(req *types.PokeRequest, verbose bool) error
	Send(req *types.PokeRequest) (*types.PokeResponse, error)
	RunBenchmark(req *types.PokeRequest, verbose bool) error
	RunSingleRequest(req *types.PokeRequest, verbose bool) error
	Route(path string, verbose bool) error
	SaveRequest(req *types.PokeRequest, saveAs string) error
	SaveResponse(resp *types.PokeResponse) error
	Load(path string) (*types.PokeRequest, error)
}

type DefaultRequestRunnerImpl struct{}

func (r *DefaultRequestRunnerImpl) Execute(req *types.PokeRequest, verbose bool) error {
	if req.Repeat > 1 {
		return r.RunBenchmark(req, verbose)
	}
	return r.RunSingleRequest(req, verbose)
}

func (r *DefaultRequestRunnerImpl) RunSingleRequest(req *types.PokeRequest, verbose bool) error {
	start := time.Now()
	resp, err := r.Send(req)
	if err != nil {
		util.Error("Request failed", err)
	}
	if err := r.SaveResponse(resp); err != nil {
		util.Error("Failed to save response", err)
	}
	duration := time.Since(start).Seconds()
	if err != nil {
		util.Error("Request failed", err)
	}
	defer resp.Raw.Body.Close()

	if req.Assert != nil {
		ok, err := util.AssertResponse(resp, req.Assert)
		if !ok {
			util.Error("Assertion failed", err)
		}
	}

	if verbose {
		util.PrintResponseVerbose(resp, req, resp.Body, duration)
	} else {
		fmt.Printf("%s\n\n", util.ColorStatus(resp.StatusCode))
		util.PrintBody(resp.Body, resp.ContentType)
	}
	return nil
}

func (r *DefaultRequestRunnerImpl) RunBenchmark(req *types.PokeRequest, verbose bool) error {
	bm := &DefaultBenchmarkerImpl{}
	res := bm.Run(req, verbose)
	_ = res
	return nil
}

func (r *DefaultRequestRunnerImpl) Send(req *types.PokeRequest) (*types.PokeResponse, error) {
	client := &http.Client{}
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBufferString(req.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
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

func (r *DefaultRequestRunnerImpl) SaveRequest(req *types.PokeRequest, path string) error {
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

func (r *DefaultRequestRunnerImpl) SaveResponse(resp *types.PokeResponse) error {
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

func (r *DefaultRequestRunnerImpl) Route(path string, verbose bool) error {
	resolver := &DefaultPayloadResolverImpl{}

	if strings.HasSuffix(path, ".json") {
		if _, err := os.Stat(path); err == nil {
			req, err := r.Load(path)
			if err != nil {
				return fmt.Errorf("failed to load request: %w", err)
			}
			body, err := resolver.Resolve(req.Body, req.BodyFile, req.BodyStdin, false)
			if err != nil {
				return fmt.Errorf("failed to resolve payload: %w", err)
			}
			req.Body = body
			req.BodyFile = ""
			req.BodyStdin = false
			return r.Execute(req, verbose)
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

		payloadResolver := &DefaultPayloadResolverImpl{}
		body, err := payloadResolver.Resolve(req.Body, req.BodyFile, req.BodyStdin, false)
		if err != nil {
			fmt.Printf("Failed to resolve body for '%s': %v\n", path, err)
			continue
		}
		req.Body = body
		req.BodyFile = ""
		req.BodyStdin = false

		err = r.Execute(req, verbose)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
		}
		fmt.Println()
	}
	return nil
}

func (r *DefaultRequestRunnerImpl) Load(path string) (*types.PokeRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var req types.PokeRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}

	if len(req.QueryParams) > 0 && !strings.Contains(req.URL, "?") {
		query := "?"
		for k, v := range req.QueryParams {
			query += fmt.Sprintf("%s=%s&", k, v)
		}
		query = strings.TrimSuffix(query, "&")
		req.URL += query
	}

	templater := &DefaultTemplateEngineImpl{}
	templater.LoadHistory()
	templater.ApplyToRequest(&req)
	return &req, nil
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
