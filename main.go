package main

import (
	"log"
	"net/http"
	"github.com/kfcoding-terminal-controller/config"
	"github.com/kfcoding-terminal-controller/client"
	"github.com/kfcoding-terminal-controller/handler"
)

func main() {
	// apiHandler, err := handler.CreateHTTPAPIHandler(client.NewClientManager("/root/.kube/config", "https://10.19.18.166:6443"))
	apiHandler, err := handler.CreateHTTPAPIHandler(client.NewClientManager("", ""))
	if err != nil {
		log.Fatal(err)
	}

	handler.Init()

	http.Handle("/api/", apiHandler)
	http.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
	// http.Handle("/", http.FileServer(http.Dir("/home/wsl/Go/src/github.com/websocket-server-shell/ui/static/")))

	log.Fatal(http.ListenAndServe(config.ServerAddress, nil))
}
