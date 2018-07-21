package service

import (
	"github.com/kfcoding-terminal-controller/types"
	"github.com/shortid"
	"github.com/kfcoding-terminal-controller/config"
)

type TerminalService struct {
	K8sService     *K8sService
	EtcdService    *EtcdService
	SessionService *SessionService
}

func (service *TerminalService) Create(body *types.TerminalBody) (string, error) {

	hostname := shortid.MustGenerate()
	podName := "terminal-" + hostname

	sessionId, err := service.SessionService.CreateSession(podName)
	if nil != err {
		return "", err
	}
	err = service.EtcdService.PutSessionId(sessionId)
	if nil != err {
		service.SessionService.DeleteSession(sessionId)
		return "", err
	}
	err = service.K8sService.CreateTerminal(body, hostname, podName)
	if nil != err {
		service.SessionService.DeleteSession(sessionId)
		service.EtcdService.DeleteSessionId(sessionId)
		return "", err
	}

	return sessionId, err
}

func (service *TerminalService) Delete(sessionId string, source int) {
	session, ok := service.SessionService.terminalSessions[sessionId]
	if source == config.SourceEtcd { // etcd删除回调
		if ok && !session.Connected { //如果session存在且没有被连接
			service.K8sService.DeleteTerminal(session.PodName)
			service.SessionService.DeleteSession(sessionId)
		}
	} else if source == config.SourceClose { //断开连接
		service.K8sService.DeleteTerminal(session.PodName)
		service.SessionService.DeleteSession(sessionId)
		//会触发etcd删除操作，导致再次调用此函数
		service.EtcdService.DeleteSessionId(sessionId)
	}
}
