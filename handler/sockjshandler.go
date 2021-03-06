package handler

import (
	"net/http"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"github.com/kfcoding-terminal-controller/service"
)

func CreateAttachHandler(service *service.SessionService, path string) http.Handler {
	return sockjs.NewHandler(path, sockjs.DefaultOptions, service.HandleTerminalSession)
}
