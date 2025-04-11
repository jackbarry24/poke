package core

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"poke/types"
	"poke/util"
)

type Benchmarker interface {
	Run(req *types.PokeRequest, verbose bool) types.BenchmarkResult
}

type DefaultBenchmarkerImpl struct{}

func (b *DefaultBenchmarkerImpl) Run(req *types.PokeRequest, verbose bool) types.BenchmarkResult {
	requestRunner := &DefaultRequestRunnerImpl{}

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
				resp, err := requestRunner.Send(req)
				duration := time.Since(t0)

				reqNum := atomic.AddInt64(&counter, 1)
				if verbose {
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

	printBenchmarkResults(result, totalTime.Seconds(), req)
	return result
}

func printBenchmarkResults(res types.BenchmarkResult, totalTime float64, req *types.PokeRequest) {
	fmt.Println()
	fmt.Println("╭──────────── Poke Benchmark ────────────╮")
	fmt.Printf("│ Requests       %-23d │\n", res.Total)
	fmt.Printf("│ Success        %-32s │\n", util.ColorString(fmt.Sprintf("%d", res.Successes), "green"))
	fmt.Printf("│ Failures       %-32s │\n", util.ColorString(fmt.Sprintf("%d", res.Failures), "red"))
	fmt.Printf("│ Total time     %-.2fs%18s │\n", totalTime, "")

	if len(res.Durations) == 0 {
		fmt.Printf("│ Avg duration   %-23s │\n", "N/A")
		fmt.Printf("│ Min            %-23s │\n", "N/A")
		fmt.Printf("│ Max            %-23s │\n", "N/A")
	} else {
		min, max := res.Durations[0], res.Durations[0]
		var sum time.Duration
		for _, d := range res.Durations {
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
			sum += d
		}
		avg := sum / time.Duration(len(res.Durations))
		fmt.Printf("│ Avg duration   %-23v │\n", avg)
		fmt.Printf("│ Min            %-32s │\n", util.ColorString(min.String(), "blue"))
		fmt.Printf("│ Max            %-32s │\n", util.ColorString(max.String(), "yellow"))
	}

	throughput := float64(res.Total) / totalTime
	fmt.Printf("│ Throughput     %-.2f req/s%12s │\n", throughput, "")
	fmt.Printf("│ Workers        %-23d │\n", req.Workers)
	fmt.Println("╰────────────────────────────────────────╯")
}
