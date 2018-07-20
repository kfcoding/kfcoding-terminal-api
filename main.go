package main

import (
	"log"
	"net/http"
	"github.com/kfcoding-terminal-controller/config"
)

func main() {

	//handler.Init()
	//
	//apiHandler, err := handler.CreateHTTPAPIHandler()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//http.Handle("/api/", apiHandler)
	//http.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
	http.Handle("/", http.FileServer(http.Dir("/Users/wsl/Go/src/github.com/kfcoding-terminal-controller/ui/static/")))

	log.Fatal(http.ListenAndServe(config.ServerAddress, nil))
}
