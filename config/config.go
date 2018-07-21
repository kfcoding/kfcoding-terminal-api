package config

import "time"

const (
	TerminalWaaAddr = "http://controller.terminal.kfcoding.com"
	ServerAddress   = "0.0.0.0:8080"
)

var (
	Version   = "v1"
	Token     = ""
	Namespace = "kfcoding-alpha"
)

var (
	KeeperPrefix = "/kfcoding/keepalive/terminal"
	KeeperTTL    = 10
)

var (
	SourceEtcd  = 0
	SourceClose = 1
)

var (
	EtcdEndPoints  = []string{"http://localhost:2379"}
	EtcdUsername   = ""
	EtcdPassword   = ""
	RequestTimeout = 10 * time.Second
)
