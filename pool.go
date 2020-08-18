// Copyright 2020 Jayden Lie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type JobFunc func(args ...interface{}) interface{}

type Job struct {
	id   int64
	fun  JobFunc
	args []interface{}
	pipe chan interface{}
}

type GoPool struct {
	LastID   int64
	PoolSize int
	Cond     *sync.Cond
	Queue    *list.List
	JobPool  *sync.Pool
}

// Coroutine pool worker process function
// @param pool: coroutine pool object
func routine(pool *GoPool) {
	for {
		pool.Cond.L.Lock()

		// First: Get job from queue front

	checkAgain:
		elem := pool.Queue.Front()
		if elem == nil {
			pool.Cond.Wait()
			goto checkAgain
		}

		job := elem.Value.(*Job)

		// Second: Remove job from queue

		pool.Queue.Remove(elem)

		pool.Cond.L.Unlock()

		job.pipe <- job.fun(job.args...) // Third: Call job process function and return value

		pool.JobPool.Put(job)
	}
}

// Create new coroutine pool object
// @param size: how many worker coroutine would be create
func NewGoPool(size int) *GoPool {
	lock := &sync.Mutex{}

	pool := &GoPool{
		LastID:   0,
		PoolSize: size,
		Cond:     sync.NewCond(lock),
		Queue:    list.New(),
		JobPool: &sync.Pool{
			New: func() interface{} {
				return &Job{}
			},
		},
	}

	pool.Cond.L.Lock() // First: stop all worker coroutine

	for i := 0; i < size; i++ {
		go routine(pool)
	}

	pool.Cond.L.Unlock() // Second: start all worker coroutine

	return pool
}

// Send a job to coroutine pool and wait for process
// @param handler: job process function handler
// @param param: job process function parameter
// @return: chan interface{}
func (pool *GoPool) Do(fun JobFunc, args ...interface{}) <-chan interface{} {
	job := pool.JobPool.Get().(*Job)

	job.Init(atomic.AddInt64(&pool.LastID, 1), fun, args)

	pool.Cond.L.Lock()
	pool.Queue.PushBack(job) // First: push job to queue
	pool.Cond.Signal()       // Second: signal worker routine
	pool.Cond.L.Unlock()

	return job.pipe
}

func (j *Job) Init(id int64, fun JobFunc, args []interface{}) {
	j.id = id
	j.fun = fun
	j.args = args
	j.pipe = make(chan interface{}, 1)
}
