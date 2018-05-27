package handler

import (
	"net/http"
	"github.com/emicklei/go-restful"
	clientapi "github.com/websocket-server-shell/client/api"
	kdErrors "github.com/websocket-server-shell/errors"
	"k8s.io/client-go/tools/remotecommand"
	"github.com/websocket-server-shell/config"
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

	// http://120.132.94.141:9090/api/sockjs?' + response.id
	response.WriteHeaderAndEntity(http.StatusOK, config.TERMINAL_WSS_ADDR+"/api/sockjs?"+sessionId)
}
