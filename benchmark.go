// A simple benchmark tool for testing web performance
// Copyright 2020 Jayden Lie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type BenchmarkItem struct {
	URL     string
	Headers map[string]string
	Params  map[string]string
	Method  string
	Body    []byte
}

type BenchmarkArgs struct {
	Simple    *BenchmarkItem
	WaitGroup *sync.WaitGroup
	Stats     *Stats
}

const (
	version = "1.0.0"
)

var (
	scriptFile     string
	targetLink     string
	logPath        string
	reqMethod      = "GET"
	reqHeaders     = make(map[string]string)
	reqArgs        = make(map[string]string)
	reqBody        []byte
	connections    = 10
	benchmarkTimes = 1
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

	if len(simple.Body) > 0 {
		opts = append(opts, BodyOption(simple.Body))
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

	stats.AddTotalPreReqs(1000000000000 / elapsed)
	stats.AddTotalReqs()
	stats.AddTotalTime(elapsed)

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

func showBenchmarkResult(stats *Stats) {
	// Make sure dividend not zero
	totalReqs := stats.totalReqs
	if totalReqs == 0 {
		totalReqs = 1
	}

	totalSeconds := stats.totalTimes / 1000
	if totalSeconds == 0 {
		totalSeconds = 1
	}

	var (
		totalRecv float64
		totalUnit string
	)

	if stats.totalRecvBytes > 1024*1024*1024 {
		totalRecv = float64(stats.totalRecvBytes) / 1024 / 1024 / 1024
		totalUnit = "GB"
	} else if stats.totalRecvBytes > 1024*1024 {
		totalRecv = float64(stats.totalRecvBytes) / 1024 / 1024
		totalUnit = "MB"
	} else if stats.totalRecvBytes > 1024 {
		totalRecv = float64(stats.totalRecvBytes) / 1024
		totalUnit = "KB"
	} else {
		totalRecv = float64(stats.totalRecvBytes)
		totalUnit = "B"
	}

	fmt.Printf("  Connections(Routines): %d\n", connections)
	fmt.Printf("  Success Total: %d reqs\n", stats.success)
	fmt.Printf("  Failure Total: %d reqs\n", stats.failure)
	fmt.Printf("  Success Rate: %d%%\n", stats.success*100/totalReqs)
	fmt.Printf("  Receive Data %0.3f(%s)\n", totalRecv, totalUnit)
	fmt.Printf("  Fastest Request: %d(MS)\n", stats.minReqElapsed)
	fmt.Printf("  Slowest Request: %d(MS)\n", stats.maxReqElapsed)
	fmt.Printf("  Average Request Time: %d(MS)\n", stats.totalTimes/totalReqs)
	fmt.Printf("  Requests/sec: %d\n", stats.totalPreReqs/stats.totalReqs/1000000)
	fmt.Printf("  Transfer/sec: %0.3f(%s)\n", totalRecv/float64(totalSeconds), totalUnit)
	fmt.Printf("----------------------------\n")

	showStatusCount(stats)
}

func parseArgs() {
	argsLen := len(os.Args)

	for i := 0; i < argsLen; i++ {
		if os.Args[i][0] == '-' && len(os.Args[i]) > 1 {
			switch os.Args[i][1] {
			case 't':
				if argsLen > i+1 {
					targetLink = os.Args[i+1]
					if len(targetLink) > 0 {
						info := strings.Split(targetLink, "://")
						if len(info) < 2 {
							targetLink = "http://" + targetLink
						}
					}
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
						benchmarkTimes = value
						i++
					}
				}
			case 'm':
				if argsLen > i+1 {
					switch strings.ToUpper(os.Args[i+1]) {
					case "GET":
						reqMethod = "GET"
					case "POST":
						reqMethod = "POST"
					}
					i++
				}
			case 'A':
				if argsLen > i+1 {
					var args map[string]string
					if err := json.Unmarshal([]byte(os.Args[i+1]), &args); err == nil {
						reqArgs = args
					}
					i++
				}
			case 'H':
				if argsLen > i+1 {
					var headers map[string]string
					if err := json.Unmarshal([]byte(os.Args[i+1]), &headers); err == nil {
						reqHeaders = headers
					}
					i++
				}
			case 'B':
				if argsLen > i+1 {
					reqBody = []byte(os.Args[i+1])
					i++
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

func startBenchmark(simples []*BenchmarkItem) {
	group := &sync.WaitGroup{}
	stats := NewStats()
	pool := NewGoPool(connections)

	for _, simple := range simples {
		group.Add(1)
		pool.Do(benchmark, NewBenchmarkArgs(simple, group, stats))
	}

	group.Wait()

	showBenchmarkResult(stats)
}

func usage() {
	fmt.Println("Usage: gobenchmark <options>           \n",
		"  Options:                                     \n",
		"    -t <S>  Testing target URL                 \n",
		"    -c <N>  Connections to keep open           \n",
		"    -n <N>  How many request for testing       \n",
		"    -L <S>  Error log path                     \n",
		"    -m <S>  Request method (etc: GET, POST)    \n",
		"    -H <S>  Add header to request (JSON format)\n",
		"    -A <S>  Request arguments (JSON format)    \n",
		"    -B <S>  Request body                       \n",
		"                                               \n",
		"    -s <S>  Load Lua script file               \n",
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

	for i := 0; i < benchmarkTimes; i++ {
		simples = append(simples, &BenchmarkItem{
			URL:     targetLink,
			Headers: reqHeaders,
			Params:  reqArgs,
			Method:  reqMethod,
			Body:    reqBody,
		})
	}

	startBenchmark(simples)
}
