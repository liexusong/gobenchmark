// Copyright 2020 Jayden Lie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"sync"
	"sync/atomic"
)

type Stats struct {
	success int64
	failure int64

	elapsedMutex  sync.Mutex
	maxReqElapsed int64
	minReqElapsed int64

	totalTimes int64
	totalReqs  int64

	totalRecvBytes int64
	totalPreReqs   int64

	statusMutex sync.Mutex
	statusStats map[int]int64
}

func NewStats() *Stats {
	return &Stats{
		statusStats: make(map[int]int64),
	}
}

func (s *Stats) AddSuccess() {
	atomic.AddInt64(&s.success, 1)
}

func (s *Stats) AddFailure() {
	atomic.AddInt64(&s.failure, 1)
}

func (s *Stats) AddTotalTime(ts int64) {
	atomic.AddInt64(&s.totalTimes, ts)
}

func (s *Stats) AddTotalPreReqs(reqs int64) {
	atomic.AddInt64(&s.totalPreReqs, reqs)
}

func (s *Stats) AddTotalReqs() {
	atomic.AddInt64(&s.totalReqs, 1)
}

func (s *Stats) AddTotalRecvBytes(bytes int64) {
	atomic.AddInt64(&s.totalRecvBytes, bytes)
}

func (s *Stats) UpdateReqElapsed(elapsed int64) {
	s.elapsedMutex.Lock()
	if s.maxReqElapsed < elapsed {
		s.maxReqElapsed = elapsed
	}
	if s.minReqElapsed == 0 || elapsed < s.minReqElapsed {
		s.minReqElapsed = elapsed
	}
	s.elapsedMutex.Unlock()
}

func (s *Stats) AddStatusCount(status int) {
	s.statusMutex.Lock()
	if _, exists := s.statusStats[status]; !exists {
		s.statusStats[status] = 0
	}
	s.statusStats[status]++
	s.statusMutex.Unlock()
}
