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
	"strconv"
	"sort"
)

type BenchmarkItem struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Params  map[string]string `json:"params"`
	Method  string            `json:"method"`
	Times   int               `json:"times"`
}

type BenchmarkArgs struct {
	Simple    *BenchmarkItem
	WaitGroup *sync.WaitGroup
}

var (
	success int64
	failure int64

	elapsedMutex  sync.Mutex
	maxReqElapsed int64
	minReqElapsed int64

	totalTimes int64
	totalReqs  int64

	totalRecvBytes int64

	statusLock  = sync.Mutex{}
	statusStats = make(map[int]int64)

	benchmarkFile = "./simples.json"
	connections   = 10
)

func statsHttpStatus(status int) {
	statusLock.Lock()

	if _, exists := statusStats[status]; !exists {
		statusStats[status] = 0
	}

	statusStats[status]++

	statusLock.Unlock()
}

func updateElapsedStats(elapsed int64) {
	elapsedMutex.Lock()

	if elapsed > maxReqElapsed {
		maxReqElapsed = elapsed
	}

	if minReqElapsed == 0 || elapsed < minReqElapsed {
		minReqElapsed = elapsed
	}

	elapsedMutex.Unlock()
}

func benchmark(args interface{}) interface{} {
	var benchArgs = args.(*BenchmarkArgs)

	if benchArgs.WaitGroup == nil {
		return nil
	}

	defer benchArgs.WaitGroup.Done()

	if benchArgs.Simple == nil {
		return nil
	}

	simple := benchArgs.Simple

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

	statsHttpStatus(req.Status)

	if err != nil || req.Status != http.StatusOK {
		atomic.AddInt64(&failure, 1)
		return nil
	}

	atomic.AddInt64(&totalRecvBytes, int64(len(rsp)))

	updateElapsedStats(elapsed)

	atomic.AddInt64(&success, 1)

	return nil
}

func displayStatusStats() {
	var status []int

	for state, _ := range statusStats {
		status = append(status, state)
	}

	sort.Ints(status)

	for _, state := range status {
		fmt.Printf("Status %d: %d reqs\n", state, statusStats[state])
	}
}

func displayBenchmarkResult() {
	fmt.Printf("\nBenchmark Result:\n")
	fmt.Printf("-------------------------------\n")
	fmt.Printf("Connections(GoRoutines): %d\n", connections)
	fmt.Printf("Success Total: %d reqs\n", success)
	fmt.Printf("Failure Total: %d reqs\n", failure)
	fmt.Printf("Success Rate: %d%%\n", success*100/totalReqs)
	fmt.Printf("Receive Data %d KB\n", totalRecvBytes/1024)
	fmt.Printf("Fastest Request: %dms\n", minReqElapsed)
	fmt.Printf("Slowest Request: %dms\n", maxReqElapsed)
	fmt.Printf("Average Request Time: %dms\n", totalTimes/totalReqs)
	fmt.Printf("-------------------------------\n")
	displayStatusStats()
}

func parseArgs() {
	argsLen := len(os.Args)

	for i := 0; i < argsLen; i++ {
		if os.Args[i][0] == '-' && len(os.Args[i]) > 1 {
			switch os.Args[i][1] {
			case 'f':
				if argsLen > i+1 {
					benchmarkFile = os.Args[i+1]
					i++
				}
			case 'c':
				if argsLen > i+1 {
					value, err := strconv.Atoi(os.Args[i+1])
					if err == nil && value > 0 {
						connections = value
						i++
					}
				}
			}
		}
	}
}

func main() {
	parseArgs()

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

	connPool := NewGoPool(connections)

	wg := sync.WaitGroup{}

	for _, simple := range simples {
		times := simple.Times
		if times <= 0 {
			times = 1
		}

		for i := 0; i < times; i++ {
			wg.Add(1)

			connPool.Do(benchmark, &BenchmarkArgs{
				Simple:    simple,
				WaitGroup: &wg,
			})
		}
	}

	wg.Wait()

	displayBenchmarkResult()
}
