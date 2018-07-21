package main

import (
	"log"
	"net/http"
	"github.com/kfcoding-terminal-controller/config"
	"github.com/kfcoding-terminal-controller/handler"
	"github.com/kfcoding-terminal-controller/service"
	"github.com/kfcoding-terminal-controller/service/common"
	"path"
)

func main() {

	k8sClient := common.InitK8sClient()

	var terminalService *service.TerminalService
	terminalService.EtcdService = service.GetEtcdService(terminalService.Delete)
	terminalService.K8sService = service.GetK8sService(k8sClient)
	terminalService.SessionService = service.GetSerssionService(k8sClient, terminalService.Delete)

	go terminalService.EtcdService.WatchSessionId(path.Join(config.KeeperPrefix, config.Version))

	apiHandler, err := handler.CreateHTTPAPIHandler(terminalService)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/api/", apiHandler)
	http.Handle("/api/sockjs/", handler.CreateAttachHandler(terminalService, "/api/sockjs"))
	http.Handle("/", http.FileServer(http.Dir("/Users/wsl/Go/src/github.com/kfcoding-terminal-controller/ui/static/")))

	log.Println("Start terminal server")
	log.Fatal(http.ListenAndServe(config.ServerAddress, nil))
}
