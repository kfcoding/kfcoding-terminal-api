package main

import (
	"log"
	"net/http"
	"github.com/kfcoding-terminal-controller/config"
	"github.com/kfcoding-terminal-controller/service/common"
	"github.com/kfcoding-terminal-controller/service"
	"path"
	"github.com/kfcoding-terminal-controller/handler"
)

func main() {

	config.InitEnv()
	var k8sClient *common.K8sClient
	if config.InCluster != "" {
		k8sClient = common.InitInClusterK8sClient()
	} else {
		k8sClient = common.InitOutClusterK8sClient()
	}

	etcdClient := common.GetMyEtcdClient()
	etcdService := service.GetEtcdService(etcdClient)
	k8sService := service.GetK8sService(k8sClient)
	sessionService := service.GetSerssionService(k8sClient)

	terminalService := &service.TerminalService{
		EtcdService:    etcdService,
		K8sService:     k8sService,
		SessionService: sessionService,
	}

	etcdService.SetOnDeleteCallback(terminalService.Delete)
	sessionService.SetOnCloseCallback(terminalService.Delete)

	go etcdService.WatchSessionId(path.Join(config.KeeperPrefix, config.Version))

	apiHandler, err := handler.CreateHTTPAPIHandler(terminalService)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/api/", apiHandler)
	http.Handle("/api/sockjs/", handler.CreateAttachHandler(sessionService, "/api/sockjs"))
	http.Handle("/", http.FileServer(http.Dir("/Users/wsl/Go/src/github.com/kfcoding-terminal-controller/ui/static/")))

	log.Println("Start terminal server")
	log.Fatal(http.ListenAndServe(config.ServerAddress, nil))
}
