package common

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"log"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultQPS         = 1e6
	DefaultBurst       = 1e6
	DefaultContentType = "application/vnd.kubernetes.protobuf"
	DefaultUserAgent   = "dashboard"
	Version            = "UNKNOWN"
)

func GetClientAndConfig() (kubernetes.Interface, *rest.Config, error) {
	cfg, err := Config()
	if err != nil {
		return nil, nil, err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return client, cfg, nil
}

func Config() (*rest.Config, error) {
	cmdConfig, err := ClientCmdConfig()
	if err != nil {
		return nil, err
	}

	cfg, err := cmdConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	initConfig(cfg)
	return cfg, nil
}

func ClientCmdConfig() (clientcmd.ClientConfig, error) {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: "/Users/wsl/.kube/config"},
		&clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{Server: "https://139.196.71.90:6443"}}).ClientConfig()

	if err != nil {
		log.Print("ClientCmdConfig error: ", err)
		return nil, err
	}

	defaultAuthInfo := buildAuthInfoFromConfig(cfg)
	authInfo := &defaultAuthInfo

	return buildCmdConfig(authInfo, cfg), nil
}

func buildCmdConfig(authInfo *api.AuthInfo, cfg *rest.Config) clientcmd.ClientConfig {
	cmdCfg := api.NewConfig()
	cmdCfg.Clusters["kubernetes"] = &api.Cluster{
		Server:                   cfg.Host,
		CertificateAuthority:     cfg.TLSClientConfig.CAFile,
		CertificateAuthorityData: cfg.TLSClientConfig.CAData,
		InsecureSkipTLSVerify:    cfg.TLSClientConfig.Insecure,
	}
	cmdCfg.AuthInfos["kubernetes"] = authInfo
	cmdCfg.Contexts["kubernetes"] = &api.Context{
		Cluster:  "kubernetes",
		AuthInfo: "kubernetes",
	}
	cmdCfg.CurrentContext = "kubernetes"

	return clientcmd.NewDefaultClientConfig(
		*cmdCfg,
		&clientcmd.ConfigOverrides{},
	)
}

func buildAuthInfoFromConfig(cfg *rest.Config) api.AuthInfo {
	return api.AuthInfo{
		Token:                 cfg.BearerToken,
		ClientCertificate:     cfg.CertFile,
		ClientKey:             cfg.KeyFile,
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
		Username:              cfg.Username,
		Password:              cfg.Password,
	}
}

func initConfig(cfg *rest.Config) {
	cfg.QPS = DefaultQPS
	cfg.Burst = DefaultBurst
	cfg.ContentType = DefaultContentType
	cfg.UserAgent = DefaultUserAgent + "/" + Version
}
