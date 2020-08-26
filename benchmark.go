// A simple benchmark tool for testing web performance
// Copyright 2020 Jayden Lie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BenchmarkItem struct {
	URL     string
	Headers map[string]string
	Params  map[string]string
	Method  string
}

type BenchmarkArgs struct {
	Simple    *BenchmarkItem
	WaitGroup *sync.WaitGroup
	Stats     *Stats
}

const (
	version = "1.0.1"
)

var (
	scriptFile     string
	targetLink     string
	logPath        string
	connections    = 10
	benchmarkCount = 1
	benchmarkTimes = 1
	intervalSecond = 1

	connPool *GoPool
)

func benchmark(params ...interface{}) interface{} {
	if len(params) <= 0 {
		return nil
	}

	var args = params[0].(*BenchmarkArgs)

	if args.WaitGroup == nil {
		return nil
	}

	defer args.WaitGroup.Done()

	if args.Simple == nil || args.Stats == nil {
		return nil
	}

	simple := args.Simple
	stats := args.Stats

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

	if !ReqRunScript(req) {
		Errorf("Call script request() function return false")
		return nil
	}

	if len(req.opts.URL) == 0 {
		Errorf("Testing target URL has not set")
		return nil
	}

	body, err := req.Do()

	elapsed := req.GetLastElapsed()

	stats.AddTotalTime(elapsed)
	stats.AddTotalReqs()

	if req.Status != 0 {
		stats.AddStatusCount(req.Status)
	}

	if err != nil || req.Status != http.StatusOK {
		stats.AddFailure()
		if err != nil {
			Errorf("%s", err.Error())
		}
		return nil
	}

	stats.AddTotalRecvBytes(int64(len(body)))
	stats.UpdateReqElapsed(elapsed)

	if CheckRunScript(body) {
		stats.AddSuccess()
	} else {
		stats.AddFailure()
		Errorf("Check result false: %s, %s", req.opts.URL, string(body))
	}

	return nil
}

func showStatusCount(stats *Stats) {
	var codes []int

	for state, _ := range stats.statusStats {
		codes = append(codes, state)
	}

	sort.Ints(codes)

	for _, code := range codes {
		fmt.Printf("Status %d: %d reqs\n", code, stats.statusStats[code])
	}
}

func showBenchmarkResult(times int, stats *Stats) {
	// Make sure dividend not zero
	totalReqs := stats.totalReqs
	if totalReqs == 0 {
		totalReqs = 1
	}

	fmt.Printf("\n     Benchmark Times(%d):\n", times)
	fmt.Printf("-------------------------------\n")
	fmt.Printf("  Connections(Routines): %d\n", connections)
	fmt.Printf("  Success Total: %d reqs\n", stats.success)
	fmt.Printf("  Failure Total: %d reqs\n", stats.failure)
	fmt.Printf("  Success Rate: %d%%\n", stats.success*100/totalReqs)
	fmt.Printf("  Receive Data %d KB\n", stats.totalRecvBytes/1024)
	fmt.Printf("  Fastest Request: %dms\n", stats.minReqElapsed)
	fmt.Printf("  Slowest Request: %dms\n", stats.maxReqElapsed)
	fmt.Printf("  Average Request Time: %dms\n", stats.totalTimes/totalReqs)
	fmt.Printf("-------------------------------\n")

	showStatusCount(stats)
}

func parseArgs() {
	argsLen := len(os.Args)

	for i := 0; i < argsLen; i++ {
		if os.Args[i][0] == '-' && len(os.Args[i]) > 1 {
			switch os.Args[i][1] {
			case 'l':
				if argsLen > i+1 {
					targetLink = os.Args[i+1]
					i++
				}
			case 'L':
				if argsLen > i+1 {
					logPath = os.Args[i+1]
					i++
				}
			case 's':
				if argsLen > i+1 {
					scriptFile = os.Args[i+1]
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
			case 'n':
				if argsLen > i+1 {
					value, err := strconv.Atoi(os.Args[i+1])
					if err == nil && value > 0 {
						benchmarkCount = value
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
			case 'h':
				usage()
				os.Exit(0)
			case 'v':
				fmt.Printf("gobenchmark version: %s\n", version)
				os.Exit(0)
			}
		}
	}
}

func NewBenchmarkArgs(simple *BenchmarkItem, group *sync.WaitGroup, stats *Stats) *BenchmarkArgs {
	return &BenchmarkArgs{
		Simple:    simple,
		WaitGroup: group,
		Stats:     stats,
	}
}

func startBenchmark(simples []*BenchmarkItem, times int) {
	group := &sync.WaitGroup{}
	stats := NewStats()

	for _, simple := range simples {
		group.Add(1)
		connPool.Do(benchmark, NewBenchmarkArgs(simple, group, stats))
	}

	group.Wait()

	showBenchmarkResult(times, stats)
}

func usage() {
	fmt.Println("Usage: gobenchmark <options>           \n",
		"  Options:                                     \n",
		"    -l <S>  Testing target URL                 \n",
		"    -c <N>  Connections to keep open           \n",
		"    -n <N>  How many request for testing       \n",
		"    -t <N>  How many times for testing         \n",
		"    -i <N>  Interval for each testing(seconds) \n",
		"    -L <S>  Error log path                     \n",
		"                                               \n",
		"    -s <S>  Load Lua script file               \n",
		"    -H <H>  Add header to request              \n",
		"    -h      Show usage for gobenchmark         \n",
		"    -v      Print version details              ")
}

func main() {
	parseArgs()

	if len(scriptFile) > 0 {
		err := InitScript(scriptFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}

	if len(logPath) > 0 {
		InitDefaultLog(logPath, DebugLevel)
	}

	var simples []*BenchmarkItem

	for i := 0; i < benchmarkCount; i++ {
		simples = append(simples, &BenchmarkItem{
			URL:     targetLink,
			Headers: nil,
			Params:  nil,
			Method:  "GET",
		})
	}

	connPool = NewGoPool(connections)

	for i := 0; i < benchmarkTimes; i++ {
		startBenchmark(simples, i+1)
		if i < benchmarkTimes-1 {
			time.Sleep(time.Duration(intervalSecond) * time.Second)
		}
	}
}
