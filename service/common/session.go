package common

import (
	"k8s.io/client-go/tools/remotecommand"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"fmt"
	"io"
	"encoding/json"
	"log"
)

type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

type TerminalMessage struct {
	Op, Data, SessionID string
	Rows, Cols          uint16
}

type TerminalSession struct {
	Id            string
	Bound         chan error
	SockJSSession sockjs.Session
	SizeChan      chan remotecommand.TerminalSize
	PodName       string
	Connected     bool
}

func (t TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.SizeChan:
		return &size
	}
}

func (t TerminalSession) Read(p []byte) (int, error) {
	m, err := t.SockJSSession.Recv()
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
		t.SizeChan <- remotecommand.TerminalSize{msg.Cols, msg.Rows}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown message type '%s'", msg.Op)
	}
}

func (t TerminalSession) Write(p []byte) (int, error) {
	if err := t.SockJSSession.Send(string(p)); err != nil {
		return 0, err
	}
	//msg, err := json.Marshal(TerminalMessage{
	//	Op:   "stdout",
	//	Data: string(p),
	//})
	//if err != nil {
	//	return 0, err
	//}

	//if err = t.SockJSSession.Send(string(msg)); err != nil {
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
	if err = t.SockJSSession.Send(string(msg)); err != nil {
		return err
	}
	return nil
}

func (t TerminalSession) Close(status uint32, reason string) {
	log.Print("terminal ", t.Id, " close , ", status, ", ", reason)
	t.SockJSSession.Close(status, reason)
}
