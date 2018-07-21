package handler

import (
	"net/http"
	"github.com/emicklei/go-restful"
	"github.com/kfcoding-terminal-controller/config"
	"log"
	types2 "github.com/kfcoding-terminal-controller/types"
	"github.com/kfcoding-terminal-controller/service"
	"strings"
)

type APIHandler struct {
	terminalService *service.TerminalService
}

func CreateHTTPAPIHandler(terminalService *service.TerminalService) (http.Handler, error) {

	apiHandler := APIHandler{
		terminalService: terminalService,
	}

	apiV1Ws := new(restful.WebService)
	apiV1Ws.Path("/api/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	// http://cloudware.wss.kfcoding.com/api/v1/pod/kfcoding-alpha/{pod}/shell/application
	apiV1Ws.Route(
		apiV1Ws.GET("/pod/{namespace}/{pod}/shell/{container}").
			To(apiHandler.HandleNewTerminal))

	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)
	wsContainer.Add(apiV1Ws)
	return wsContainer, nil
}

func (apiHandler *APIHandler) HandleNewTerminal(request *restful.Request, response *restful.Response) {
	if !apiHandler.checkToken(request, response) {
		return
	}
	body := &types2.TerminalBody{}
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

	sessionId, err := apiHandler.terminalService.Create(body)

	response.Header().Set("Access-Control-Allow-Origin", "*")
	if err == nil {
		// http://120.132.94.141:9090/api/sockjs?' + response.id
		log.Print("HandleNewTerminal ok: ", config.TerminalWaaAddr+"/api/sockjs?"+sessionId)
		response.WriteHeaderAndEntity(http.StatusOK, types2.ResponseBody{Data: config.TerminalWaaAddr + "/api/sockjs?" + sessionId})
	} else {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, types2.ResponseBody{Error: err.Error()})
	}

	//sessionId, err := genTerminalSessionId()
	//if err != nil {
	//	log.Print("handleExecShell error: ", err)
	//	response.WriteHeaderAndEntity(http.StatusInternalServerError, types2.ResponseBody{Error: err.Error()})
	//	return
	//}
	//
	//podName := request.PathParameter("pod")
	//lock.Lock()
	//terminalSessions[sessionId] = TerminalSession{
	//	id:       sessionId,
	//	bound:    make(chan error),
	//	sizeChan: make(chan remotecommand.TerminalSize),
	//	pod:      podName,
	//}
	//lock.Unlock()
	//
	//go WaitForTerminal(request, sessionId)

	// http://120.132.94.141:9090/api/sockjs?' + response.id
	//response.Write([]byte(config.TerminalWaaAddr + "/api/sockjs?" + sessionId))
}

func (apiHandler *APIHandler) checkToken(request *restful.Request, response *restful.Response) bool {
	token := request.HeaderParameter("Token")
	if strings.Compare(token, config.Token) != 0 {
		log.Print("认证失败")
		response.WriteHeaderAndEntity(http.StatusUnauthorized, types2.ResponseBody{Error: "认证失败"})
		return false
	}
	return true
}
