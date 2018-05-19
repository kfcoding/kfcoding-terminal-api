package main

import (
	"log"
	"net/http"
	"github.com/kfcoding-shell-server/client"
	"github.com/kfcoding-shell-server/handler"
)

func main() {
	apiHandler, err := handler.CreateHTTPAPIHandler(client.NewClientManager("/root/.kube/config", "https://10.19.18.166:6443"))
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/api/", apiHandler)
	http.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
	http.Handle("/", http.FileServer(http.Dir("/Users/wsl/Go/src/github.com/kfcoding-shell-server/ui/static/")))

	go func() { log.Fatal(http.ListenAndServe("0.0.0.0:9090", nil)) }()
	select {}
}
