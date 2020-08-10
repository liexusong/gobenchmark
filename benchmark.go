// A simple benchmark tool for testing web performance

package main

import (
	"encoding/json"
	"os"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"net/http"
	"strconv"
	"sort"
	"time"
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
	benchmarkFile = "./simples.json"
	connections   = 10

	benchmarkTimes = 1
	intervalSecond = 1

	stats *Stats
)

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

	stats.AddTotalTime(elapsed)
	stats.AddTotalReqs()

	stats.AddStatusCount(req.Status)

	if err != nil || req.Status != http.StatusOK {
		stats.AddFailure()
		return nil
	}

	stats.AddTotalRecvBytes(int64(len(rsp)))
	stats.UpdateReqElapsed(elapsed)
	stats.AddSuccess()

	return nil
}

func displayStatusStats() {
	var status []int

	for state, _ := range stats.statusStats {
		status = append(status, state)
	}

	sort.Ints(status)

	for _, state := range status {
		fmt.Printf("Status %d: %d reqs\n", state, stats.statusStats[state])
	}
}

func displayBenchmarkResult(times int) {
	fmt.Printf("\n       Benchmark(%d):\n", times)
	fmt.Printf("-------------------------------\n")
	fmt.Printf("  Connections(GoRoutines): %d\n", connections)
	fmt.Printf("  Success Total: %d reqs\n", stats.success)
	fmt.Printf("  Failure Total: %d reqs\n", stats.failure)
	fmt.Printf("  Success Rate: %d%%\n", stats.success*100/stats.totalReqs)
	fmt.Printf("  Receive Data %d KB\n", stats.totalRecvBytes/1024)
	fmt.Printf("  Fastest Request: %dms\n", stats.minReqElapsed)
	fmt.Printf("  Slowest Request: %dms\n", stats.maxReqElapsed)
	fmt.Printf("  Average Request Time: %dms\n", stats.totalTimes/stats.totalReqs)
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
			case 't':
				if argsLen > i+1 {
					value, err := strconv.Atoi(os.Args[i+1])
					if err == nil && value > 0 {
						benchmarkTimes = value
						i++
					}
				}
			case 'i':
				if argsLen > i+1 {
					value, err := strconv.Atoi(os.Args[i+1])
					if err == nil && value > 0 {
						intervalSecond = value
						i++
					}
				}
			}
		}
	}
}

func startBenchmark(simples []*BenchmarkItem, times int) {
	stats = NewStats()

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

	displayBenchmarkResult(times)
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

	for i := 0; i < benchmarkTimes; i++ {
		startBenchmark(simples, i+1)
		if i < benchmarkTimes-1 {
			time.Sleep(time.Duration(intervalSecond)*time.Second)
		}
	}
}
