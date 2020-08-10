# GoBenchmark

#### 安装

```shell
$ go build .
```

#### 使用方式:

```shell
$ ./gobenchmark -f ./simple.json -c 100 -t 100 -i 10
```

*   `-f simple.json`：要测试的实例样本
*   `-c 100`：测试的连接数
*   `-t 100`：压测次数
*   `-i 10`：每次压测间隔多少秒

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
     Benchmark Times(1):
-------------------------------
  Connections(GoRoutines): 100
  Success Total: 1000 reqs
  Failure Total: 0 reqs
  Success Rate: 100%
  Receive Data 2185 KB
  Fastest Request: 19ms
  Slowest Request: 135ms
  Average Request Time: 77ms
-------------------------------
Status 200: 1000 reqs                           
```

