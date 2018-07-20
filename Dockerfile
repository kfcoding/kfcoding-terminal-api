FROM golang:1.10.3-alpine3.8

MAINTAINER "wsl <wsl@kfcoding.com>"

ADD ./controller /usr/bin/

EXPOSE 8080

CMD ["controller"]