// Copyright 2020 Jayden Lie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Options struct {
	URL         string
	Method      int
	Headers     map[string]string
	Params      map[string]string
	Body        []byte
	ContentType string
	Timeout     time.Duration
}

type Request struct {
	opts    *Options
	Elapsed int64
	Status  int
}

type Option func(*Options)

const (
	MethodGet  = 1
	MethodPost = 2
	MethodNone = 3
)

var (
	clientPool = sync.Pool{
		New: func() interface{} {
			return &http.Client{}
		},
	}

	skipSSLTransport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
)

func CaseCompare(src, dst string) int {
	size := len(src)
	if size > len(dst) {
		size = len(dst)
	}

	offset := uint8('a' - 'A')

	for i := 0; i < size; i++ {
		c1 := src[i]
		c2 := dst[i]

		if c1 >= 'a' && c1 <= 'z' {
			c1 -= offset
		}

		if c2 >= 'a' && c2 <= 'z' {
			c2 -= offset
		}

		ret := c1 - c2
		if ret > 0 {
			return 1
		} else if ret < 0 {
			return -1
		}
	}

	return len(src) - len(dst)
}

func HasScheme(url string) bool {
	size := len(url)

	if (size > 8 && strings.ToLower(url[0:8]) == "https://") ||
		(size > 7 && strings.ToLower(url[0:7]) == "http://") {
		return true
	}

	return false
}

func NewRequest(opts ...Option) *Request {
	req := &Request{
		opts: &Options{
			Method:  MethodGet,
			Timeout: time.Second * 5,
			Params:  make(map[string]string),
			Body:    nil,
			Headers: make(map[string]string),
		},
	}

	for _, opt := range opts {
		opt(req.opts)
	}

	return req
}

func URLOption(url string) Option {
	return func(opt *Options) {
		opt.URL = url
	}
}

func MethodOption(method int) Option {
	return func(opt *Options) {
		if method >= MethodNone {
			method = MethodGet
		}
		opt.Method = method
	}
}

func HeaderOption(field, value string) Option {
	return func(opt *Options) {
		opt.Headers[field] = value
		if len(opt.ContentType) == 0 && CaseCompare(field, "Content-Type") == 0 {
			opt.ContentType = strings.ToLower(value)
		}
	}
}

func HeadersOption(headers map[string]string) Option {
	return func(opt *Options) {
		if headers == nil {
			return
		}

		for field, value := range headers {
			opt.Headers[field] = value

			if len(opt.ContentType) == 0 && CaseCompare(field, "Content-Type") == 0 {
				opt.ContentType = strings.ToLower(value)
			}
		}
	}
}

func ParamOption(field, value string) Option {
	return func(opt *Options) {
		opt.Params[field] = value
	}
}

func ParamsOption(params map[string]string) Option {
	return func(opt *Options) {
		if params == nil {
			return
		}

		for field, value := range params {
			opt.Params[field] = value
		}
	}
}

func BodyOption(body []byte) Option {
	return func(opt *Options) {
		opt.Body = body
	}
}

func TimeoutOption(timeout time.Duration) Option {
	return func(opt *Options) {
		opt.Timeout = timeout
	}
}

func (req *Request) encodeURI() string {
	var uri string

	if len(req.opts.Params) > 0 {
		for field, value := range req.opts.Params {
			uri = fmt.Sprintf("%s&%s=%s", uri, field, value)
		}
		uri = strings.TrimLeft(uri, "&")
	}

	return uri
}

func (req *Request) get(client *http.Client) (*http.Response, error) {
	url := req.opts.URL

	if len(req.opts.Params) > 0 {
		url = fmt.Sprintf("%s?%s", url, req.encodeURI())
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for field, value := range req.opts.Headers {
		request.Header[field] = []string{value}
	}

	rsp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	req.Status = rsp.StatusCode

	return rsp, nil
}

func (req *Request) post(client *http.Client) (*http.Response, error) {
	var body []byte

	if req.opts.Body != nil {
		body = req.opts.Body
	} else if req.opts.ContentType == "application/json" && len(req.opts.Params) > 0 {
		body, _ = json.Marshal(req.opts.Params)
	} else {
		body = []byte(req.encodeURI())
	}

	request, err := http.NewRequest("POST", req.opts.URL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	for field, value := range req.opts.Headers {
		request.Header[field] = []string{value}
	}

	rsp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	req.Status = rsp.StatusCode

	return rsp, nil
}

func getTimestampMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (req *Request) Do() ([]byte, error) {
	if len(req.opts.URL) < 7 {
		return nil, errors.New("request URL cannot be empty")
	}

	client := clientPool.Get().(*http.Client)

	if req.opts.Timeout > 0 {
		client.Timeout = req.opts.Timeout
	}

	if strings.ToLower(req.opts.URL[0:5]) == "https" {
		client.Transport = skipSSLTransport
	} else {
		client.Transport = http.DefaultTransport
	}

	var (
		rsp *http.Response
		err error
	)

	sTime := getTimestampMs()

	switch req.opts.Method {
	case MethodGet:
		rsp, err = req.get(client)
	case MethodPost:
		rsp, err = req.post(client)
	default:
		err = errors.New("unsupported method")
	}

	req.Elapsed = getTimestampMs() - sTime

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rsp.Body.Close()

		clientPool.Put(client)
	}()

	return ioutil.ReadAll(rsp.Body)
}

func (req *Request) GetLastElapsed() int64 {
	return req.Elapsed
}

func (req *Request) GetLastStatus() int {
	return req.Status
}

func (req *Request) SetURL(url string) {
	url = strings.TrimSpace(url)
	if len(url) == 0 {
		return
	}

	if !HasScheme(url) {
		url = "http://" + url
	}

	req.opts.URL = url
}

func (req *Request) SetHeader(field, value string) {
	req.opts.Headers[field] = value
}

func (req *Request) SetParam(field, value string) {
	req.opts.Params[field] = value
}

func (req *Request) SetBody(body []byte) {
	req.opts.Body = body
}

func (req *Request) SetMethod(method string) {
	switch method {
	case "POST":
		req.opts.Method = MethodPost
	case "GET":
		req.opts.Method = MethodGet
	}
}

func (req *Request) SetTimeout(ms int64) {
	req.opts.Timeout = time.Duration(ms) * time.Millisecond
}
