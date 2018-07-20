package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/emicklei/go-restful"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sync"
	"time"
)

type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

type TerminalSession struct {
	id            string
	bound         chan error
	sockJSSession sockjs.Session
	sizeChan      chan remotecommand.TerminalSize
	pod           string
}

type TerminalMessage struct {
	Op, Data, SessionID string
	Rows, Cols          uint16
}

func (t TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	}
}

func (t TerminalSession) Read(p []byte) (int, error) {
	m, err := t.sockJSSession.Recv()
	if err != nil {
		return 0, err
	}

	var msg TerminalMessage
	if err := json.Unmarshal([]byte(m), &msg); err != nil {
		return 0, err
	}

	switch msg.Op {
	case "stdin":
		return copy(p, msg.Data), nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{msg.Cols, msg.Rows}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown message type '%s'", msg.Op)
	}
}

func (t TerminalSession) Write(p []byte) (int, error) {
	if err := t.sockJSSession.Send(string(p)); err != nil {
		return 0, err
	}
	//msg, err := json.Marshal(TerminalMessage{
	//	Op:   "stdout",
	//	Data: string(p),
	//})
	//if err != nil {
	//	return 0, err
	//}

	//if err = t.sockJSSession.Send(string(msg)); err != nil {
	//	return 0, err
	//}
	return len(p), nil
}

func (t TerminalSession) Toast(p string) error {
	msg, err := json.Marshal(TerminalMessage{
		Op:   "toast",
		Data: p,
	})
	if err != nil {
		return err
	}

	if err = t.sockJSSession.Send(string(msg)); err != nil {
		return err
	}
	return nil
}

func (t TerminalSession) Close(status uint32, reason string) {
	t.sockJSSession.Close(status, reason)

	lock.Lock()
	delete(terminalSessions, t.id)
	lock.Unlock()

	log.Print(t.id, " , ", status, ", ", reason)

	DeleteTerminal(t.pod)
}

var terminalSessions = make(map[string]TerminalSession)
var lock sync.Mutex

func handleTerminalSession(session sockjs.Session) {
	var (
		buf             string
		err             error
		msg             TerminalMessage
		terminalSession TerminalSession
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
	if terminalSession, ok = terminalSessions[msg.SessionID]; !ok {
		log.Printf("handleTerminalSession: can't find session '%s'", msg.SessionID)
		return
	}

	terminalSession.sockJSSession = session

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("recover", err)
		}
	}()
	terminalSession.bound <- nil

	terminalSessions[msg.SessionID] = terminalSession

	log.Print("new connection : ", msg.SessionID)
}

func CreateAttachHandler(path string) http.Handler {
	return sockjs.NewHandler(path, sockjs.DefaultOptions, handleTerminalSession)
}

func startProcess(request *restful.Request, cmd []string, ptyHandler PtyHandler) error {
	namespace := request.PathParameter("namespace")
	podName := request.PathParameter("pod")
	containerName := request.PathParameter("container")

	log.Print("namespace: " + namespace)
	log.Print("podname: " + podName)
	log.Print("containerName:" + containerName)
	req := K8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	var exec remotecommand.Executor
	var err error

	for i := 0; i < 5; i++ {
		exec, err = remotecommand.NewSPDYExecutor(Config, "POST", req.URL())
		if err != nil {
			log.Print("sleep, remotecommand.NewSPDYExecutor: ", err)
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}

	if err != nil {
		log.Print("return, remotecommand.NewSPDYExecutor error: ", err)
		return err
	}

	log.Print("before exec.Stream ")

	for i := 0; i < 5; i++ {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:             ptyHandler,
			Stdout:            ptyHandler,
			Stderr:            ptyHandler,
			TerminalSizeQueue: ptyHandler,
			Tty:               true,
		})
		if err != nil {
			log.Print("sleep, : exec.Stream", err)
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	if err != nil {
		log.Print("return,  exec.Stream: ", err)
		return err
	}

	return nil
}

func genTerminalSessionId() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	id := make([]byte, hex.EncodedLen(len(bytes)))
	hex.Encode(id, bytes)
	return string(id), nil
}

func isValidShell(validShells []string, shell string) bool {
	for _, validShell := range validShells {
		if validShell == shell {
			return true
		}
	}
	return false
}

func WaitForTerminal(request *restful.Request, sessionId string) {
	shell := request.QueryParameter("shell")

	select {
	case <-terminalSessions[sessionId].bound:

		close(terminalSessions[sessionId].bound)

		var err error
		validShells := []string{"sh", "cmd", "bash", "powershell"}

		if isValidShell(validShells, shell) {
			cmd := []string{shell}
			err = startProcess(request, cmd, terminalSessions[sessionId])
		} else {
			// No shell given or it was not valid: try some shells until one succeeds or all fail
			// FIXME: if the first shell fails then the first keyboard event is lost
			for _, testShell := range validShells {
				cmd := []string{testShell}
				if err = startProcess(request, cmd, terminalSessions[sessionId]); err == nil {
					break
				}
			}
		}

		if err != nil {
			terminalSessions[sessionId].Close(2, err.Error())
			return
		}

		terminalSessions[sessionId].Close(1, "Process exited")
	}
}
