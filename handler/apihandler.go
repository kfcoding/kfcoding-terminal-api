package handler

import (
	"net/http"
	"github.com/emicklei/go-restful"
	clientapi "github.com/kfcoding-shell-server/client/api"
	kdErrors "github.com/kfcoding-shell-server/errors"
	"k8s.io/client-go/tools/remotecommand"
	"log"
)

type APIHandler struct {
	cManager clientapi.ClientManager
}

type TerminalResponse struct {
	Id string `json:"id"`
}

func CreateHTTPAPIHandler(cManager clientapi.ClientManager) (http.Handler, error) {

	apiHandler := APIHandler{cManager: cManager}
	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)

	apiV1Ws := new(restful.WebService)

	apiV1Ws.Path("/api/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	wsContainer.Add(apiV1Ws)

	apiV1Ws.Route(
		apiV1Ws.GET("/pod/{namespace}/{pod}/shell/{container}").
			To(apiHandler.handleExecShell).
			Writes(TerminalResponse{}))

	return wsContainer, nil
}

func (apiHandler *APIHandler) handleExecShell(request *restful.Request, response *restful.Response) {

	log.Println("before", len(terminalSessions))

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

	terminalSessions[sessionId] = TerminalSession{
		id:       sessionId,
		bound:    make(chan error),
		sizeChan: make(chan remotecommand.TerminalSize),
	}

	log.Println("after", len(terminalSessions))

	go WaitForTerminal(k8sClient, cfg, request, sessionId)

	response.Header().Set("Access-Control-Allow-Origin", "*")

	response.WriteHeaderAndEntity(http.StatusOK, TerminalResponse{Id: sessionId})
}
