package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type BenchmarkResult struct {
	Total     int
	Successes int
	Failures  int
	Durations []time.Duration
}

func RunBenchmark(req *PokeRequest, verbose bool) BenchmarkResult {
	var wg sync.WaitGroup
	resultChan := make(chan time.Duration, req.Repeat)
	errorChan := make(chan bool, req.Repeat)

	startTime := time.Now()

	// Calculate the base workload per worker and the remainder
	baseWorkload := req.Repeat / req.Workers
	remainder := req.Repeat % req.Workers

	var requestCounter int64

	for i := range req.Workers {
		wg.Add(1)
		go func(workerIndex int) {
			defer wg.Done()
			// Assign an extra request to the first 'remainder' workers
			workload := baseWorkload
			if workerIndex < remainder {
				workload++
			}
			for j := 0; j < workload; j++ {
				t0 := time.Now()
				resp, err := SendRequest(*req)
				duration := time.Since(t0)

				reqNum := atomic.AddInt64(&requestCounter, 1)
				if verbose {
					status := colorStatus(resp.StatusCode)
					fmt.Printf("Request %-3d [Worker %-2d]: %-7s (%v)\n", reqNum, workerIndex, status, duration)
				}

				if err != nil {
					errorChan <- true
					continue
				}
				resp.Body.Close()

				if req.ExpectStatus > 0 && resp.StatusCode != req.ExpectStatus {
					errorChan <- true
					continue
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
	successes := 0
	failures := 0

	for ok := range errorChan {
		if ok {
			failures++
		} else {
			successes++
		}
	}
	for d := range resultChan {
		durations = append(durations, d)
	}

	res := BenchmarkResult{
		Total:     req.Repeat,
		Successes: successes,
		Failures:  failures,
		Durations: durations,
	}
	printBenchmarkResults(res, totalTime, req)
	return res
}

func printBenchmarkResults(res BenchmarkResult, totalTime time.Duration, req *PokeRequest) {
	fmt.Println()
	fmt.Println("===================================")
	fmt.Println("       --- Poke Benchmark ---      ")
	fmt.Println("===================================")
	fmt.Printf("Requests:      %d\n", res.Total)
	fmt.Printf("Success:       %d\n", res.Successes)
	fmt.Printf("Failures:      %d\n", res.Failures)
	fmt.Printf("Total time:    %.2fs\n", totalTime.Seconds())

	if len(res.Durations) == 0 {
		fmt.Println("Avg duration:  N/A")
		fmt.Println("Min:           N/A")
		fmt.Println("Max:           N/A")
	} else {
		min, max, sum := res.Durations[0], res.Durations[0], time.Duration(0)
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
		fmt.Printf("Avg duration:  %v\n", avg)
		fmt.Printf("Min:           %v\n", min)
		fmt.Printf("Max:           %v\n", max)
	}

	throughput := float64(res.Total) / totalTime.Seconds()
	fmt.Printf("Throughput:    %.2f req/sec\n", throughput)
	fmt.Printf("Workers:       %d\n", req.Workers)
	fmt.Println("===================================")
}
