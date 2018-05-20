package main

import (
	"log"
	"net/http"
	"github.com/websocket-server-shell/handler"
	"github.com/websocket-server-shell/client"
)

func main() {
	//apiHandler, err := handler.CreateHTTPAPIHandler(client.NewClientManager("/root/.kube/config", "https://10.19.18.166:6443"))
	apiHandler, err := handler.CreateHTTPAPIHandler(client.NewClientManager("", ""))
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/api/", apiHandler)
	http.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
	//http.Handle("/", http.FileServer(http.Dir("/home/wsl/Go/src/github.com/websocket-server-shell/ui/static/")))

	go func() { log.Fatal(http.ListenAndServe("0.0.0.0:9090", nil)) }()
	select {}
}
