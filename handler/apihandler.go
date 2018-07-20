package handler

import (
	"net/http"
	"github.com/emicklei/go-restful"
	clientapi "github.com/kfcoding-terminal-controller/client/api"
	kdErrors "github.com/kfcoding-terminal-controller/errors"
	"k8s.io/client-go/tools/remotecommand"
	"github.com/kfcoding-terminal-controller/config"
	"log"
	types2 "github.com/kfcoding-terminal-controller/types"
)

type APIHandler struct {
	cManager clientapi.ClientManager
}

func CreateHTTPAPIHandler(cManager clientapi.ClientManager) (http.Handler, error) {

	apiHandler := APIHandler{
		cManager: cManager,
	}
	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)

	apiV1Ws := new(restful.WebService)

	apiV1Ws.Path("/api/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	wsContainer.Add(apiV1Ws)

	apiV1Ws.Route(
		apiV1Ws.POST("/pod").
			To(apiHandler.handleCreateTerminal))
	// http://cloudware.wss.kfcoding.com/api/v1/pod/kfcoding-alpha/{pod}/shell/application
	apiV1Ws.Route(
		apiV1Ws.GET("/pod/{namespace}/{pod}/shell/{container}").
			To(apiHandler.handleExecShell))

	return wsContainer, nil
}

func (apiHandler *APIHandler) handleExecShell(request *restful.Request, response *restful.Response) {

	sessionId, err := genTerminalSessionId()
	if err != nil {
		kdErrors.HandleInternalError(response, err)
		return
	}

	k8sClient, err := apiHandler.cManager.Client(request)
	if err != nil {
		kdErrors.HandleInternalError(response, err)
		return
	}

	cfg, err := apiHandler.cManager.Config(request)
	if err != nil {
		kdErrors.HandleInternalError(response, err)
		return
	}

	podName := request.PathParameter("pod")
	lock.Lock()
	terminalSessions[sessionId] = TerminalSession{
		id:       sessionId,
		bound:    make(chan error),
		sizeChan: make(chan remotecommand.TerminalSize),
		pod:      podName,
	}
	lock.Unlock()

	go WaitForTerminal(k8sClient, cfg, request, sessionId)

	response.Header().Set("Access-Control-Allow-Origin", "*")

	log.Print(config.TerminalWaaAddr + "/api/sockjs?" + sessionId)

	// http://120.132.94.141:9090/api/sockjs?' + response.id
	response.Write([]byte(config.TerminalWaaAddr + "/api/sockjs?" + sessionId))
}

func (apiHandler *APIHandler) handleCreateTerminal(request *restful.Request, response *restful.Response) {
	if !apiHandler.checkToken(request, response) {
		return
	}
	body := types2.TerminalBody{}
	if err := request.ReadEntity(body); nil != err {
		log.Print("handleCreateTerminal error: ", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, types2.ResponseBody{Error: err.Error()})
		return
	}
	if body.Image == "" {
		log.Print("handleCreateTerminal error: Image 不能为空")
		response.WriteHeaderAndEntity(http.StatusInternalServerError, types2.ResponseBody{Error: "Image 不能为空"})
		return
	}
	log.Printf("handleCreateTerminal: %+v\n", body)

	data, err := CreateTerminal(&body)

	if err == nil {
		response.WriteHeaderAndEntity(http.StatusOK, types2.ResponseBody{Data: data})
	} else {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, types2.ResponseBody{Error: err.Error()})
	}
}

func (apiHandler *APIHandler) checkToken(request *restful.Request, response *restful.Response) bool {
	//token := request.HeaderParameter("Token")
	//if strings.Compare(token, configs.Token) != 0 {
	//	log.Print("认证失败")
	//	response.WriteHeaderAndEntity(http.StatusUnauthorized, types.ResponseBody{Error: "认证失败"})
	//	return false
	//} else {
	//	log.Print("认证成功")
	//}
	return true
}
