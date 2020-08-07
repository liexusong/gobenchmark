// Copyright 2020 Jayden Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type JobFunc func(interface{}) interface{}

type Job struct {
	id   int64
	fun  JobFunc
	arg  interface{}
	pipe chan interface{}
}

type GoPool struct {
	LastID   int64
	PoolSize int
	Cond     *sync.Cond
	Lock     *sync.Mutex
	Queue    *list.List
}

// Coroutine pool worker process function
// @param pool: coroutine pool object
func routine(pool *GoPool) {
	for {
		pool.Lock.Lock()

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

		pool.Lock.Unlock()

		job.pipe <- job.fun(job.arg) // Third: Call job process function and return value
	}
}

// Create new coroutine pool object
// @param size: how many worker coroutine would be create
func New(size int) *GoPool {
	lock := &sync.Mutex{}

	pool := &GoPool{
		LastID:   0,
		PoolSize: size,
		Cond:     sync.NewCond(lock),
		Lock:     lock,
		Queue:    list.New(),
	}

	pool.Lock.Lock() // Stop all worker coroutine

	for i := 0; i < size; i++ {
		go routine(pool)
	}

	pool.Lock.Unlock() // Start all worker coroutine

	return pool
}

// Send a job to coroutine pool and wait for process
// @param handler: job process function handler
// @param param: job process function parameter
// @return: chan interface{}
func (pool *GoPool) Do(fun JobFunc, arg interface{}) <-chan interface{} {
	job := &Job{
		id:   atomic.AddInt64(&pool.LastID, 1),
		fun:  fun,
		arg:  arg,
		pipe: make(chan interface{}, 1),
	}

	pool.Lock.Lock()
	pool.Queue.PushBack(job) // Push job to queue
	pool.Cond.Signal()       // Signal worker routine
	pool.Lock.Unlock()

	return job.pipe
}
