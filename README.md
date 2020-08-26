# GoBenchmark

#### 安装

```shell
$ go build .
```

#### 使用方式:

```shell
Usage: gobenchmark <options>
   Options:
     -l <S>  Testing target URL
     -c <N>  Connections to keep open
     -n <N>  How many request for testing
     -t <N>  How many times for testing
     -i <N>  Interval for each testing(seconds)
     -L <S>  Error log path
     -m <S>  Request method (etc: GET, POST)
     -H <S>  Add header to request (JSON format)
     -A <S>  Request arguments (JSON format)
     -B <S>  Request body

     -s <S>  Load Lua script file
     -h      Show usage for gobenchmark
     -v      Print version details
```

```shell
$ ./gobenchmark -l http://testing-url -c 100 -t 100 -i 10 -s ./script/script.lua
```

*   `-l http://testing-url`：要测试的目标URL
*   `-s ./script/script.lua`：测试脚本
*   `-c 100`：测试的连接数
*   `-t 100`：压测次数
*   `-i 10`：每次压测间隔多少秒
*   `-L ./error.log`：如果请求出错，会在这里记录日志

#### 测试脚本

测试脚本是一个lua脚本，这个脚本必须提供3个函数：`init()`、`request()` 和 `check()`。

* `init()`：测试时仅此调用一次，一般用于初始化一些测试的数据。
* `request()`：每次请求测试URL都会调用这个函数，一般用于设置请求的参数。
* `check()`：每次请求测试完毕都会调用一次，可以用于检测结果是否正确。

这3个函数都需要返回一个bool值，表示调用是否成功。

#### 测试结果：

```
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

     Benchmark Times(2):
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

...
```

