## 终端控制器（暂不使用）

- 根据容器镜像名创建一个终端并返回websocket地址

- 客户端根据websocket地址连接到终端

1. build

```
go build -o controller main.go

build using docker golang

docker run -it -v /Users/wsl/go/src:/go/src golang:1.10.3-alpine3.8 sh

cd src/github.com/kfcoding-terminal-controller/ && go build -o controller main.go && exit

scp controller root@worker:/home/kfcoding-terminal-controller

cd /home/kfcoding-terminal-controller && \

docker build -t daocloud.io/shaoling/kfcoding-terminal-controller:v1.7 .
```

2. 创建Termianl

```
POST /api/v1/terminal

Header
    Content-Type: application/json
    Token ""

Body
    {
        "Image":"ubuntu"
    }

Response
    {
        Data:""
        Error:""
    }
Image:  要启动的容器镜像名称
Data:   Websocket地址
```

3. Websocket(sockjs)连接

```
GET /api/sockjs/
```
