package config

import (
	"time"
	"os"
	"strings"
	"strconv"
	"log"
)

var (
	InCluster       = ""
	Version         = "v1"
	Token           = ""
	Namespace       = "kfcoding-alpha"
	ServerAddress   = "0.0.0.0:8080"
	TerminalWaaAddr = "http://controller.terminal.kfcoding.com"
)

var (
	KeeperPrefix = "/kfcoding/keepalive/terminal"
	KeeperTTL    = 60
)

var (
	EtcdEndPoints  = []string{"http://localhost:2379"}
	EtcdUsername   = ""
	EtcdPassword   = ""
	RequestTimeout = 10 * time.Second
)

const (
	SourceEtcd  = 0
	SourceClose = 1
)

func InitEnv() {

	if inCluster := os.Getenv("InCluster"); inCluster != "" {
		InCluster = inCluster
	}
	if version := os.Getenv("Version"); version != "" {
		Version = version
	}
	if token := os.Getenv("Token"); token != "" {
		Token = token
	}
	if namespace := os.Getenv("Namespace"); namespace != "" {
		Namespace = namespace
	}
	if serverAddress := os.Getenv("ServerAddress"); serverAddress != "" {
		ServerAddress = serverAddress
	}
	if terminalWaaAddr := os.Getenv("TerminalWaaAddr"); terminalWaaAddr != "" {
		TerminalWaaAddr = terminalWaaAddr
	}

	// etcd config
	if etcdEndPoint := os.Getenv("EtcdEndPoints"); "" != etcdEndPoint {
		EtcdEndPoints = strings.Split(etcdEndPoint, ",")
	}
	if etcdUsername := os.Getenv("EtcdUsername"); "" != etcdUsername {
		EtcdUsername = etcdUsername
	}
	if etcdPassword := os.Getenv("EtcdPassword"); "" != etcdPassword {
		EtcdPassword = etcdPassword
	}

	// keep alive config
	if ttl := os.Getenv("KeeperTTL"); "" != ttl {
		if t, err := strconv.ParseInt(ttl, 10, 64); nil != err {
			log.Fatal(err)
		} else {
			KeeperTTL = int(t)
		}
	}
	if keeperPrefix := os.Getenv("KeeperPrefix"); keeperPrefix != "" {
		KeeperPrefix = keeperPrefix
	}

	log.Print("InCluster:        ", InCluster)
	log.Print("Version:          ", Version)
	log.Print("Token:            ", Token)
	log.Print("Namespace:        ", Namespace)
	log.Print("ServerAddress:    ", ServerAddress)
	log.Print("TerminalWaaAddr:  ", TerminalWaaAddr)

	log.Print("KeeperTTL:        ", KeeperTTL)
	log.Print("KeeperPrefix:     ", KeeperPrefix)

	log.Print("EtcdEndPoints:    ", EtcdEndPoints)
	log.Print("EtcdUsername:     ", EtcdUsername)
	log.Print("EtcdPassword:     ", EtcdPassword)

}
