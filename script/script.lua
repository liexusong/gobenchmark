mark = require "gobenchmark"
json = require "./script/json"

-- 启动测试时会被调用一次(可以初始化一些请求参数)
function init()
    rsp, res = mark.curl("www.baidu.com")
    print(rsp)
    return true
end

-- 每个请求都会被调用一次(请求前: 可以设置请求的一些参数)
function request(req)
    req:set_timeout(10000)                 -- 设置超时时间(毫秒)
    req:set_header("host", "yourhost.com") -- 设置header
    return true
end

-- 每个请求都会被调用一次(请求后: 检测返回数据是否正确)
function check(rsp)
    -- result = json.decode(rsp)
    -- 检测返回数据是否正确
    return true
end
