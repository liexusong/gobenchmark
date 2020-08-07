// A simple benchmark tool for testing web performance

package main

import (
	"encoding/json"
	"sync/atomic"
	"os"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"net/http"
)

type RspStatus struct {
	Status string `json:"status"`
}

var (
	success int64
	failure int64

	failureErrors int64
	failureStatus int64

	elapsedMutex  sync.Mutex
	maxReqElapsed int64
	minReqElapsed int64

	totalTimes int64
	totalReqs  int64
)

func benchmark(simple *BenchmarkItem, wg *sync.WaitGroup) {
	defer wg.Done()

	method := MethodGet
	if len(simple.Method) > 0 {
		switch strings.ToUpper(simple.Method) {
		case "GET":
			method = MethodGet
		case "POST":
			method = MethodPost
		}
	}

	var opts []Option

	opts = append(opts, URLOption(simple.URL))
	opts = append(opts, MethodOption(method))

	if len(simple.Headers) > 0 {
		opts = append(opts, HeadersOption(simple.Headers))
	}

	if len(simple.Params) > 0 {
		opts = append(opts, ParamsOption(simple.Params))
	}

	req := NewRequest(opts...)

	rsp, err := req.Do()

	elapsed := req.GetLastElapsed()

	atomic.AddInt64(&totalTimes, elapsed)
	atomic.AddInt64(&totalReqs, 1)

	if err != nil || req.Status != http.StatusOK {
		atomic.AddInt64(&failure, 1)
		atomic.AddInt64(&failureErrors, 1)
		return
	}

	rspStatus := &RspStatus{}

	if err := json.Unmarshal(rsp, rspStatus); err != nil || strings.ToLower(rspStatus.Status) != "ok" {
		atomic.AddInt64(&failure, 1)
		atomic.AddInt64(&failureStatus, 1)
	} else {
		elapsedMutex.Lock()
		if elapsed > maxReqElapsed {
			maxReqElapsed = elapsed
		}
		if minReqElapsed == 0 || elapsed < minReqElapsed {
			minReqElapsed = elapsed
		}
		elapsedMutex.Unlock()

		atomic.AddInt64(&success, 1)
	}
}

func displayBenchmarkResult() {
	fmt.Printf("Benchmark Result:\n")
	fmt.Printf("-----------------\n")
	fmt.Printf("Success Total: %d reqs\n", success)
	fmt.Printf("Failure Total: %d reqs, Service Errors: %d, Status Errors: %d\n", failure, failureErrors, failureStatus)
	fmt.Printf("Success Rate: %d%%\n", success*100/totalReqs)
	fmt.Printf("Max Elapsed Request: %d Millseconds, Min Elapsed Request: %d Millseconds\n", maxReqElapsed, minReqElapsed)
	fmt.Printf("Request Average Times: %d Millseconds\n", totalTimes/totalReqs)
}

type BenchmarkItem struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Params  map[string]string `json:"params"`
	Method  string            `json:"method"`
	Times   int               `json:"times"`
}

func main() {
	var benchmarkFile = "./simples.json"
	if len(os.Args) > 1 {
		benchmarkFile = os.Args[1]
	}

	file, err := os.Open(benchmarkFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = file.Close() }()

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	var simples []*BenchmarkItem

	if err := json.Unmarshal(buf, &simples); err != nil {
		fmt.Println(err)
		return
	}

	wg := sync.WaitGroup{}

	for _, simple := range simples {
		times := simple.Times
		if times <= 0 {
			times = 1
		}

		for i := 0; i < times; i++ {
			wg.Add(1)
			go benchmark(simple, &wg)
		}
	}

	wg.Wait()

	displayBenchmarkResult()
}
