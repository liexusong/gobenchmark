# GoBenchmark

#### 安装

```shell
$ go build .
```

#### 使用方式:

```shell
$ ./gobenchmark ./simple.json
```

*   simple.json：要测试的实例样本

#### 样本格式：

```json
[
    {
        "url":"http://......",
        "headers":{
            "Content-Type":"text/json"
        },
        "params":{
            "type":"1,2,3,4,5,6"
        },
        "method": "get",
        "times": 1
    },
    {
        "url":"http://......",
        "headers":{
            "Content-Type":"text/json"
        },
        "params":{
            "type":"1,2,3,4,5,6"
        },
        "method": "post",
        "times": 10
    }
]
```

*   url：要压测的URL
*   headers：设置的HTTP头信息
*   params：传递的参数
*   method：请求的方法，支持 `POST` 和 `GET` 两种
*   times：请求次数

#### 测试结果：

```shell
Benchmark Result:
-----------------
Success Total: 990 reqs
Failure Total: 10 reqs, Service Errors: 0, Status Errors: 10
Success Rate: 99%
Max Elapsed Request: 1968 Millseconds, Min Elapsed Request: 422 Millseconds
Request Average Times: 1232 Millseconds
```

