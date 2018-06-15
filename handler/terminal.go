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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sync"
	"io/ioutil"
	"github.com/terminal-controller/config"
	"time"
)

// PtyHandler is what remotecommand expects from a pty
type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

// TerminalSession implements PtyHandler (using a SockJS connection)
type TerminalSession struct {
	id            string
	bound         chan error
	sockJSSession sockjs.Session
	sizeChan      chan remotecommand.TerminalSize
	pod           string
}

// TerminalMessage is the messaging protocol between ShellController and TerminalSession.
//
// OP      DIRECTION  FIELD(S) USED  DESCRIPTION
// ---------------------------------------------------------------------
// bind    fe->be     SessionID      Id sent back from TerminalResponse
// stdin   fe->be     Data           Keystrokes/paste buffer
// resize  fe->be     Rows, Cols     New terminal size
// stdout  be->fe     Data           Output from the process
// toast   be->fe     Data           OOB message to be shown to the user
type TerminalMessage struct {
	Op, Data, SessionID string
	Rows, Cols          uint16
}

// TerminalSize handles pty->process resize events
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	}
}

// Read handles pty->process messages (stdin, resize)
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Read(p []byte) (int, error) {

	//log.Println("read", string(p))

	m, err := t.sockJSSession.Recv()

	//log.Println("read2 ", m)
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

// Write handles process->pty stdout
// Called from remotecommand whenever there is any output
func (t TerminalSession) Write(p []byte) (int, error) {
	//log.Println("write", string(p))

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

// Toast can be used to send the user any OOB messages
// hterm puts these in the center of the terminal
func (t TerminalSession) Toast(p string) error {
	//log.Println("toast", p)
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

// Close shuts down the SockJS connection and sends the status code and reason to the client
// Can happen if the process exits or if there is an error starting up the process
// For now the status code is unused and reason is shown to the user (unless "")
func (t TerminalSession) Close(status uint32, reason string) {
	t.sockJSSession.Close(status, reason)

	lock.Lock()
	delete(terminalSessions, t.id)
	lock.Unlock()

	log.Print(t.id, " , ", status, ", ", reason)

	// call api to delete container
	url := config.API_SERVER_ADDR + "/cloudware/deleteContainer?podName=" + t.pod + "&type=1"
	req, _ := http.NewRequest("DELETE", url, nil)
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(body))
}

// terminalSessions stores a map of all TerminalSession objects
// FIXME: this structure needs locking
var terminalSessions = make(map[string]TerminalSession)
var lock sync.Mutex

// handleTerminalSession is Called by net/http for any new /api/sockjs connections
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

// CreateAttachHandler is called from main for /api/sockjs
func CreateAttachHandler(path string) http.Handler {
	return sockjs.NewHandler(path, sockjs.DefaultOptions, handleTerminalSession)
}

// startProcess is called by handleAttach
// Executed cmd in the container specified in request and connects it up with the ptyHandler (a session)
func startProcess(k8sClient kubernetes.Interface, cfg *rest.Config, request *restful.Request, cmd []string, ptyHandler PtyHandler) error {
	namespace := request.PathParameter("namespace")
	podName := request.PathParameter("pod")
	containerName := request.PathParameter("container")

	log.Print("namespace: " + namespace)
	log.Print("podname: " + podName)
	log.Print("containerName:" + containerName)

	req := k8sClient.CoreV1().RESTClient().Post().
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
		exec, err = remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
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
			log.Print("sleep, exec.Stream: ", err)
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

// genTerminalSessionId generates a random session ID string. The format is not really interesting.
// This ID is used to identify the session when the client opens the SockJS connection.
// Not the same as the SockJS session id! We can't use that as that is generated
// on the client side and we don't have it yet at this point.
func genTerminalSessionId() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	id := make([]byte, hex.EncodedLen(len(bytes)))
	hex.Encode(id, bytes)
	return string(id), nil
}

// isValidShell checks if the shell is an allowed one
func isValidShell(validShells []string, shell string) bool {
	for _, validShell := range validShells {
		if validShell == shell {
			return true
		}
	}
	return false
}

// WaitForTerminal is called from apihandler.handleAttach as a goroutine
// Waits for the SockJS connection to be opened by the client the session to be bound in handleTerminalSession
func WaitForTerminal(k8sClient kubernetes.Interface, cfg *rest.Config, request *restful.Request, sessionId string) {
	shell := request.QueryParameter("shell")

	select {
	case <-terminalSessions[sessionId].bound:

		close(terminalSessions[sessionId].bound)

		var err error
		validShells := []string{"cmd", "bash", "sh", "powershell"}

		if isValidShell(validShells, shell) {
			cmd := []string{shell}
			err = startProcess(k8sClient, cfg, request, cmd, terminalSessions[sessionId])
		} else {
			// No shell given or it was not valid: try some shells until one succeeds or all fail
			// FIXME: if the first shell fails then the first keyboard event is lost
			for _, testShell := range validShells {
				cmd := []string{testShell}
				if err = startProcess(k8sClient, cfg, request, cmd, terminalSessions[sessionId]); err == nil {
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
