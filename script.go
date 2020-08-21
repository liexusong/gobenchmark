package main

import (
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

var (
	luaState   *lua.LState
	reqHeaders = make(map[string]string)
)

func InitScript() {
	luaState = lua.NewState()

	luaState.SetGlobal("set_header", luaState.NewFunction(SetHeader))
	luaState.SetGlobal("set_headers", luaState.NewFunction(SetHeaders))
	luaState.SetGlobal("curl", luaState.NewFunction(CURL))
	luaState.SetGlobal("echo", luaState.NewFunction(Echo))
}

func RunScript(script string) error {
	return luaState.DoFile(script)
}

func SetHeader(L *lua.LState) int {
	field := L.ToString(1)
	value := L.ToString(2)

	reqHeaders[field] = value

	return 0
}

func SetHeaders(L *lua.LState) int {
	headers := L.ToTable(1)

	headers.ForEach(func(field lua.LValue, value lua.LValue) {
		reqHeaders[field.String()] = value.String()
	})

	return 0
}

// example: curl(method, url, args)
func CURL(L *lua.LState) int {
	method := L.ToString(1)
	url := L.ToString(2)
	args := L.ToTable(3)

	var (
		methodOpt int
		paramsOpt = make(map[string]string)
	)

	switch strings.ToUpper(method) {
	case "GET":
		methodOpt = MethodGet
	case "POST":
		methodOpt = MethodPost
	}

	args.ForEach(func(field lua.LValue, value lua.LValue) {
		paramsOpt[field.String()] = value.String()
	})

	req := NewRequest(
		MethodOption(methodOpt),
		URLOption(url),
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

func Echo(L *lua.LState) int {
	value := L.ToString(1)

	fmt.Println(value)

	return 0
}
