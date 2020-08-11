// A simple benchmark tool for testing web performance
// Copyright 2020 Jayden Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	Stats     *Stats
}

var (
	benchmarkFile = "./simples.json"
	connections   = 10

	benchmarkTimes = 1
	intervalSecond = 1
)

func benchmark(data interface{}) interface{} {
	var args = data.(*BenchmarkArgs)

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
	fmt.Printf("\n     Benchmark Times(%d):\n", times)
	fmt.Printf("-------------------------------\n")
	fmt.Printf("  Connections(Routines): %d\n", connections)
	fmt.Printf("  Success Total: %d reqs\n", stats.success)
	fmt.Printf("  Failure Total: %d reqs\n", stats.failure)
	fmt.Printf("  Success Rate: %d%%\n", stats.success*100/stats.totalReqs)
	fmt.Printf("  Receive Data %d KB\n", stats.totalRecvBytes/1024)
	fmt.Printf("  Fastest Request: %dms\n", stats.minReqElapsed)
	fmt.Printf("  Slowest Request: %dms\n", stats.maxReqElapsed)
	fmt.Printf("  Average Request Time: %dms\n", stats.totalTimes/stats.totalReqs)
	fmt.Printf("-------------------------------\n")

	showStatusCount()
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

func NewBenchmarkArgs(simple *BenchmarkItem, group *sync.WaitGroup, stats *Stats) *BenchmarkArgs {
	return &BenchmarkArgs {
		Simple:    simple,
		WaitGroup: group,
		Stats:     stats,
	}
}

func startBenchmark(simples []*BenchmarkItem, times int) {
	connPool := NewGoPool(connections)

	group := &sync.WaitGroup{}
	stats := NewStats()

	for _, simple := range simples {
		if simple.Times <= 0 {
			simple.Times = 1
		}

		for i := 0; i < simple.Times; i++ {
			group.Add(1)
			connPool.Do(benchmark, NewBenchmarkArgs(simple, group, stats))
		}
	}

	group.Wait()

	showBenchmarkResult(times, stats)
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
