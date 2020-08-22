// Copyright 2020 Jayden Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

const (
	reqMeta = "request"
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

// example: curl(method, url, headers, args)
func CURL(L *lua.LState) int {
	method := L.ToString(1)
	url := L.ToString(2)
	headers := L.ToTable(3)
	args := L.ToTable(4)

	var (
		methodOpt  int
		headersOpt = make(map[string]string)
		paramsOpt  = make(map[string]string)
	)

	switch strings.ToUpper(method) {
	case "GET":
		methodOpt = MethodGet
	case "POST":
		methodOpt = MethodPost
	}

	headers.ForEach(func(field lua.LValue, value lua.LValue) {
		headersOpt[field.String()] = value.String()
	})

	args.ForEach(func(field lua.LValue, value lua.LValue) {
		paramsOpt[field.String()] = value.String()
	})

	req := NewRequest(
		MethodOption(methodOpt),
		URLOption(url),
		HeadersOption(headersOpt),
		ParamsOption(paramsOpt),
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
	L.ArgError(1, "req() expected")
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
		Fn:      L.GetGlobal("req"),
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
