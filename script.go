// Copyright 2020 Jayden Lie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	reqMeta = "gobenchmark_req"
)

var (
	L         *lua.LState
	stateLock sync.Mutex

	enableLua bool

	exports = map[string]lua.LGFunction{
		"curl": CURL,
	}
)

func LoadModule(L *lua.LState) int {
	module := L.SetFuncs(L.NewTable(), exports)
	L.Push(module)
	return 1
}

func InitScript(script string) error {
	L = lua.NewState()

	L.PreloadModule("gobenchmark", LoadModule)

	err := L.DoFile(script)
	if err != nil {
		return err
	}

	RegisterReqMeta(L)

	err = L.CallByParam(lua.P{
		Fn:      L.GetGlobal("init"),
		NRet:    1,
		Protect: true,
		Handler: nil,
	})

	if err != nil {
		return err
	}

	result := L.Get(-1)
	L.Pop(1)

	if ok := result.(lua.LBool); !ok {
		return errors.New("call script init() function return false")
	}

	enableLua = true

	return nil
}

// Helper functions:
// Fetch content from remote URL
// Example: gobenchmark.curl(url, method, headers, params, timeout)
func CURL(L *lua.LState) int {
	target := L.CheckString(1)
	method := L.OptString(2, "GET")
	headers := L.OptTable(3, L.NewTable())
	params := L.OptTable(4, L.NewTable())
	timeout := L.OptInt64(5, int64(10*time.Second))

	var (
		methodOpt  = MethodGet
		headersOpt = make(map[string]string)
		paramsOpt  = make(map[string]string)
	)

	if !HasScheme(target) {
		target = "http://" + target
	}

	switch strings.ToUpper(method) {
	case "GET":
		methodOpt = MethodGet
	case "POST":
		methodOpt = MethodPost
	}

	headers.ForEach(func(field lua.LValue, value lua.LValue) {
		headersOpt[field.String()] = value.String()
	})

	params.ForEach(func(field lua.LValue, value lua.LValue) {
		paramsOpt[field.String()] = value.String()
	})

	req := NewRequest(
		MethodOption(methodOpt),
		URLOption(target),
		HeadersOption(headersOpt),
		ParamsOption(paramsOpt),
		TimeoutOption(time.Duration(timeout)),
	)

	rsp, err := req.Do()
	if err != nil {
		L.Push(lua.LString(""))
		L.Push(lua.LBool(false))
	} else {
		L.Push(lua.LString(string(rsp)))
		L.Push(lua.LBool(true))
	}

	return 2
}

func RegisterReqMeta(L *lua.LState) {
	mt := L.NewTypeMetatable(reqMeta)
	L.SetGlobal(reqMeta, mt)
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), reqMethods))
}

func GetReqMeta(L *lua.LState, req *Request) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = req
	L.SetMetatable(ud, L.GetTypeMetatable(reqMeta))
	return ud
}

func checkReq(L *lua.LState) *Request {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Request); ok {
		return v
	}
	L.ArgError(1, "checkReq() expected")
	return nil
}

func ReqSetHeader(L *lua.LState) int {
	req := checkReq(L)
	if L.GetTop() == 3 {
		req.SetHeader(L.CheckString(2), L.CheckString(3))
	}
	return 0
}

func ReqSetParam(L *lua.LState) int {
	req := checkReq(L)
	if L.GetTop() == 3 {
		req.SetParam(L.CheckString(2), L.CheckString(3))
	}
	return 0
}

func ReqSetBody(L *lua.LState) int {
	req := checkReq(L)
	if L.GetTop() == 2 {
		req.SetBody([]byte(L.CheckString(2)))
	}
	return 0
}

func ReqSetMethod(L *lua.LState) int {
	req := checkReq(L)
	if L.GetTop() == 2 {
		req.SetMethod(L.CheckString(2))
	}
	return 0
}

func ReqSetURL(L *lua.LState) int {
	req := checkReq(L)
	if L.GetTop() == 2 {
		req.SetURL(L.CheckString(2))
	}
	return 0
}

func ReqSetTimeout(L *lua.LState) int {
	req := checkReq(L)
	if L.GetTop() == 2 {
		req.SetTimeout(L.CheckInt64(2))
	}
	return 0
}

var reqMethods = map[string]lua.LGFunction{
	"set_header":  ReqSetHeader,
	"set_param":   ReqSetParam,
	"set_body":    ReqSetBody,
	"set_method":  ReqSetMethod,
	"set_url":     ReqSetURL,
	"set_timeout": ReqSetTimeout,
}

func ReqRunScript(req *Request) bool {
	if !enableLua {
		return true
	}

	result := false

	stateLock.Lock()

	err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("request"),
		NRet:    1,
		Protect: true,
		Handler: nil,
	}, GetReqMeta(L, req))

	if err == nil {
		ret := L.Get(-1)
		L.Pop(1)

		if ok := ret.(lua.LBool); ok {
			result = true
		}
	}

	stateLock.Unlock()

	return result
}

func CheckRunScript(body []byte) bool {
	if !enableLua {
		return true
	}

	result := false

	stateLock.Lock()

	err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("check"),
		NRet:    1,
		Protect: true,
		Handler: nil,
	}, lua.LString(string(body)))

	if err == nil {
		ret := L.Get(-1)
		L.Pop(1)

		if ok := ret.(lua.LBool); ok {
			result = true
		}
	}

	stateLock.Unlock()

	return result
}
