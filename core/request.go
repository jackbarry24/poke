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
	Save(req *types.PokeRequest, saveAs string) error
	Load(path string) (*types.PokeRequest, error)
}

type DefaultRequestRunnerImpl struct{}

func (r *DefaultRequestRunnerImpl) Execute(req *types.PokeRequest, verbose bool) error {
	if req.Repeat > 1 {
		return r.RunBenchmark(req, verbose)
	}
	return r.runSingleRequest(req, verbose)
}

func (r *DefaultRequestRunnerImpl) runSingleRequest(req *types.PokeRequest, verbose bool) error {
	start := time.Now()
	resp, err := r.Send(req)
	duration := time.Since(start).Seconds()
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Raw.Body.Close()

	if req.ExpectStatus > 0 && resp.StatusCode != req.ExpectStatus {
		return fmt.Errorf("unexpected status code: expected %d, got %d", req.ExpectStatus, resp.StatusCode)
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
		Header:      resp.Header,
		Body:        bodyBytes,
		ContentType: resp.Header.Get("Content-Type"),
		Raw:         resp,
	}, nil
}

func (r *DefaultRequestRunnerImpl) Save(req *types.PokeRequest, saveAs string) error {
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

	path := r.resolveSavePath(saveAs)
	return os.WriteFile(path, out, 0644)
}

func (r *DefaultRequestRunnerImpl) resolveSavePath(input string) string {
	if filepath.IsAbs(input) || strings.Contains(input, "/") {
		return input
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return input
	}
	dir := filepath.Join(home, ".poke")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, input)
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
	templater.ApplyRequest(&req, map[string]string{})
	return &req, nil
}
