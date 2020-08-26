// Copyright 2020 Jayden Lie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"time"
)

type Log struct {
	Path         string
	File         *os.File
	DisplayLevel int
}

const (
	DebugLevel = iota
	InfoLevel
	ErrorLevel
)

var (
	log       *Log
	logEnable bool
)

func InitDefaultLog(path string, displayLevel int) {
	var err error

	log, err = NewLog(path, displayLevel)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	logEnable = true
}

func NewLog(path string, displayLevel int) (*Log, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return nil, err
	}

	return &Log{
		Path:         path,
		File:         file,
		DisplayLevel: displayLevel,
	}, nil
}

func (l *Log) logFormat(level int, format string, args ...interface{}) bool {
	if level < l.DisplayLevel {
		return true
	}

	var levelPrefix string

	dateTime := time.Now().Format("2006-01-02 15:04:05")

	switch level {
	case DebugLevel:
		levelPrefix = "DEBUG"
	case InfoLevel:
		levelPrefix = "INFO"
	case ErrorLevel:
		levelPrefix = "ERROR"
	}

	content := fmt.Sprintf("[%s] <%s> %s\n", dateTime, levelPrefix, fmt.Sprintf(format, args...))

	bytes, err := l.File.Write([]byte(content))
	if err != nil || len(content) != bytes {
		return false
	}

	return true
}

func (l *Log) Debugf(format string, args ...interface{}) {
	l.logFormat(DebugLevel, format, args...)
}

func (l *Log) Infof(format string, args ...interface{}) {
	l.logFormat(InfoLevel, format, args...)
}

func (l *Log) Errorf(format string, args ...interface{}) {
	l.logFormat(ErrorLevel, format, args...)
}

func Debugf(format string, args ...interface{}) {
	if logEnable {
		log.logFormat(DebugLevel, format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	if logEnable {
		log.logFormat(InfoLevel, format, args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if logEnable {
		log.logFormat(ErrorLevel, format, args...)
	}
}
