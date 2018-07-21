package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sync"
	"time"
	"github.com/kfcoding-terminal-controller/service/common"
	"github.com/kfcoding-terminal-controller/config"
)

type OnCloseCallback func(podName string, source int)

type SessionService struct {
	terminalSessions map[string]common.TerminalSession
	lock             sync.Mutex
	k8sClient        *common.K8sClient
	onClose          OnCloseCallback
}

func GetSerssionService(k8sClient *common.K8sClient, callback OnCloseCallback) *SessionService {
	return &SessionService{
		terminalSessions: make(map[string]common.TerminalSession),
		k8sClient:        k8sClient,
		onClose:          callback,
	}
}

func (service *SessionService) CreateSession(podName string) (string, error) {
	sessionId, err := service.genTerminalSessionId()
	if nil != err {
		log.Print("CreateSession error: ", err)
		return "", err
	}
	session := common.TerminalSession{
		Id:        sessionId,
		Bound:     make(chan error),
		SizeChan:  make(chan remotecommand.TerminalSize),
		PodName:   podName,
		Connected: false,
	}
	service.lock.Lock()
	service.terminalSessions[sessionId] = session
	service.lock.Unlock()
	log.Print("CreateSession ok: ", session)

	return sessionId, err
}

func (service *SessionService) DeleteSession(sessionId string) {
	service.lock.Lock()
	delete(service.terminalSessions, sessionId)
	service.lock.Unlock()
	log.Print("DeleteSession ok: ", sessionId)
}

func (service *SessionService) HandleTerminalSession(session sockjs.Session) {
	var (
		buf             string
		err             error
		msg             common.TerminalMessage
		terminalSession common.TerminalSession
		ok              bool
	)
	if buf, err = session.Recv(); err != nil {
		log.Printf("handleTerminalSession: can't Recv: %v", err)
		return
	}
	if err = json.Unmarshal([]byte(buf), &msg); err != nil {
		log.Printf("handleTerminalSession: can't UnMarshal (%v): %s", err, buf)
		return
	}
	if msg.Op != "bind" {
		log.Printf("handleTerminalSession: expected 'bind' message, got: %s", buf)
		return
	}
	if terminalSession, ok = service.terminalSessions[msg.SessionID]; !ok {
		log.Printf("handleTerminalSession: can't find session '%s'", msg.SessionID)
		return
	}

	terminalSession.SockJSSession = session

	go service.WaitForTerminal(&terminalSession)

	//defer func() {
	//	if err := recover(); err != nil {
	//		fmt.Println("recover", err)
	//	}
	//}()
	//terminalSession.Bound <- nil

	service.terminalSessions[msg.SessionID] = terminalSession

	log.Print("new connection: ", msg.SessionID)
}

func (service *SessionService) WaitForTerminal(session *common.TerminalSession) {
	var err error
	validShells := []string{"bash", "sh", "cmd", "powershell"}

	for _, testShell := range validShells {
		cmd := []string{testShell}
		if err = service.startProcess(session, cmd); err == nil {
			break
		}
	}

	if err != nil {
		log.Print("WaitForTerminal error: ", err)
		session.Close(2, err.Error())
	} else {
		session.Close(1, "Process exited")
	}
	service.onClose(session.Id, config.SourceClose)
}

func (service *SessionService) startProcess(session *common.TerminalSession, cmd []string) error {

	log.Print("startProcess: ", session.PodName)

	req := service.k8sClient.K8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(session.PodName).
		Namespace(config.Namespace).
		SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Container: "Application",
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	var exec remotecommand.Executor
	var err error

	for i := 0; i < 5; i++ {
		exec, err = remotecommand.NewSPDYExecutor(service.k8sClient.Config, "POST", req.URL())
		if err != nil {
			log.Print("startProcess sleep, remotecommand.NewSPDYExecutor: ", err)
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}

	if err != nil {
		log.Print("startProcess return, remotecommand.NewSPDYExecutor error: ", err)
		return err
	}

	for i := 0; i < 5; i++ {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:             session,
			Stdout:            session,
			Stderr:            session,
			TerminalSizeQueue: session,
			Tty:               true,
		})
		if err != nil {
			log.Print("startProcess sleep, : exec.Stream", err)
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	if err != nil {
		log.Print("startProcess return, exec.Stream: ", err)
		return err
	}
	return nil
}

func (service *SessionService) genTerminalSessionId() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	id := make([]byte, hex.EncodedLen(len(bytes)))
	hex.Encode(id, bytes)
	return string(id), nil
}

func (service *SessionService) isValidShell(validShells []string, shell string) bool {
	for _, validShell := range validShells {
		if validShell == shell {
			return true
		}
	}
	return false
}
