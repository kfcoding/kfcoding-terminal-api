package main

import (
	"log"
	"net/http"
	"github.com/terminal-controller/config"
	"github.com/terminal-controller/client"
	"github.com/terminal-controller/handler"
)

func main() {
	// apiHandler, err := handler.CreateHTTPAPIHandler(client.NewClientManager("/root/.kube/config", "https://10.19.18.166:6443"))
	apiHandler, err := handler.CreateHTTPAPIHandler(client.NewClientManager("", ""))
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/api/", apiHandler)
	http.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
	http.Handle("/", http.FileServer(http.Dir("/home/wsl/Go/src/github.com/websocket-server-shell/ui/static/")))

	log.Fatal(http.ListenAndServe(config.SERVER_ADDRESS, nil))
}
