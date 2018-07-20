## 终端控制器

1. build

```
go build -o controller main.go

build using docker golang

docker run -it -v /Users/wsl/go/src:/go/src golang:1.10.3-alpine3.8 sh

cd src/github.com/kfcoding-terminal-controller/ && go build -o controller main.go && exit

scp controller root@worker:/home/kfcoding-terminal-controller

cd /home/kfcoding-terminal-controller && \
docker build -t daocloud.io/shaoling/kfcoding-terminal-controller:v1.1 .

```