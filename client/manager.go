package client

import (
	"crypto/rand"
	"errors"
	"log"
	"strings"

	"github.com/emicklei/go-restful"
	authApi "github.com/kfcoding-terminal-controller/auth/api"
	clientapi "github.com/kfcoding-terminal-controller/client/api"
	"k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Dashboard UI default values for client configs.
const (
	// High enough QPS to fit all expected use cases. QPS=0 is not set here, because
	// client code is overriding it.
	DefaultQPS = 1e6
	// High enough Burst to fit all expected use cases. Burst=0 is not set here, because
	// client code is overriding it.
	DefaultBurst = 1e6
	// Use kubernetes protobuf as content type by default
	DefaultContentType = "application/vnd.kubernetes.protobuf"
	// Default cluster/context/auth name to be set in clientcmd config
	DefaultCmdConfigName = "kubernetes"
	// Header name that contains token used for authorization. See TokenManager for more information.
	JWETokenHeader = "jweToken"
	// Default http header for user-agent
	DefaultUserAgent = "dashboard"
)

// VERSION of this binary
var Version = "UNKNOWN"

// clientManager implements ClientManager interface
type clientManager struct {
	// Autogenerated key on backend start used to secure requests from csrf attacks
	csrfKey string
	// Path to kubeconfig file. If both kubeConfigPath and apiserverHost are empty
	// inClusterConfig will be used
	kubeConfigPath string
	// Address of apiserver host in format 'protocol://address:port'
	apiserverHost string
	// Initialized on clientManager creation and used if kubeconfigPath and apiserverHost are
	// empty
	inClusterConfig *rest.Config
	// Responsible for decrypting tokens coming in request header. Used for authentication.
	tokenManager authApi.TokenManager
	// Kubernetes client created without providing auth info. It uses permissions granted to
	// service account used by dashboard or kubeconfig file if it was passed during dashboard init.
	insecureClient kubernetes.Interface
}

// Client returns kubernetes client that is created based on authentication information extracted
// from request. If request is nil then authentication will be skipped.
func (self *clientManager) Client(req *restful.Request) (kubernetes.Interface, error) {
	cfg, err := self.Config(req)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// InsecureClient returns kubernetes client that was created without providing auth info. It uses permissions granted
// to service account used by dashboard or kubeconfig file if it was passed during dashboard init.
func (self *clientManager) InsecureClient() kubernetes.Interface {
	return self.insecureClient
}

// CanI returns true when user is allowed to access data provided within SelfSubjectAccessReview, false otherwise.
func (self *clientManager) CanI(req *restful.Request, ssar *v1.SelfSubjectAccessReview) bool {
	// In case user is not authenticated (uses skip option) do not allow access.
	if info, _ := self.extractAuthInfo(req); info == nil {
		return false
	}

	client, err := self.Client(req)
	if err != nil {
		log.Println(err)
		return false
	}

	response, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(ssar)
	if err != nil {
		log.Println(err)
		return false
	}

	return response.Status.Allowed
}

// Config creates rest Config based on authentication information extracted from request.
// Currently request header is only checked for existence of 'Authentication: BearerToken'
func (self *clientManager) Config(req *restful.Request) (*rest.Config, error) {
	cmdConfig, err := self.ClientCmdConfig(req)
	if err != nil {
		return nil, err
	}

	cfg, err := cmdConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	self.initConfig(cfg)
	return cfg, nil
}

// ClientCmdConfig creates ClientCmd Config based on authentication information extracted from request.
// Currently request header is only checked for existence of 'Authentication: BearerToken'
func (self *clientManager) ClientCmdConfig(req *restful.Request) (clientcmd.ClientConfig, error) {
	authInfo, err := self.extractAuthInfo(req)
	if err != nil {
		return nil, err
	}

	cfg, err := self.buildConfigFromFlags(self.apiserverHost, self.kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// Use auth data provided in cfg if extracted auth info is nil
	if authInfo == nil {
		defaultAuthInfo := self.buildAuthInfoFromConfig(cfg)
		authInfo = &defaultAuthInfo
	}

	return self.buildCmdConfig(authInfo, cfg), nil
}

// CSRFKey returns key that is generated upon client manager creation
func (self *clientManager) CSRFKey() string {
	return self.csrfKey
}

// HasAccess configures K8S api client with provided auth info and executes a basic check against apiserver to see
// if it is valid.
func (self *clientManager) HasAccess(authInfo api.AuthInfo) error {
	cfg, err := self.buildConfigFromFlags(self.apiserverHost, self.kubeConfigPath)
	if err != nil {
		return err
	}

	clientConfig := self.buildCmdConfig(&authInfo, cfg)
	cfg, err = clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	_, err = client.ServerVersion()
	return err
}

// VerberClient returns new verber client based on authentication information extracted from request
func (self *clientManager) VerberClient(req *restful.Request) (clientapi.ResourceVerber, error) {
	client, err := self.Client(req)
	if err != nil {
		return nil, err
	}

	return NewResourceVerber(client.CoreV1().RESTClient(),
		client.ExtensionsV1beta1().RESTClient(), client.AppsV1beta2().RESTClient(),
		client.BatchV1().RESTClient(), client.BatchV1beta1().RESTClient(), client.AutoscalingV1().RESTClient(),
		client.StorageV1().RESTClient()), nil
}

// SetTokenManager sets the token manager that will be used for token decryption.
func (self *clientManager) SetTokenManager(manager authApi.TokenManager) {
	self.tokenManager = manager
}

// Initializes config with default values
func (self *clientManager) initConfig(cfg *rest.Config) {
	cfg.QPS = DefaultQPS
	cfg.Burst = DefaultBurst
	cfg.ContentType = DefaultContentType
	cfg.UserAgent = DefaultUserAgent + "/" + Version
}

// Returns rest Config based on provided apiserverHost and kubeConfigPath flags. If both are
// empty then in-cluster config will be used and if it is nil the error is returned.
func (self *clientManager) buildConfigFromFlags(apiserverHost, kubeConfigPath string) (
	*rest.Config, error) {
	if len(kubeConfigPath) > 0 || len(apiserverHost) > 0 {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
			&clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{Server: apiserverHost}}).ClientConfig()
	}

	if self.isRunningInCluster() {
		return self.inClusterConfig, nil
	}

	return nil, errors.New("Could not create client config. Check logs for more information")
}

// Based on rest config creates auth info structure.
func (self *clientManager) buildAuthInfoFromConfig(cfg *rest.Config) api.AuthInfo {
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

// Based on auth info and rest config creates client cmd config.
func (self *clientManager) buildCmdConfig(authInfo *api.AuthInfo, cfg *rest.Config) clientcmd.ClientConfig {
	cmdCfg := api.NewConfig()
	cmdCfg.Clusters[DefaultCmdConfigName] = &api.Cluster{
		Server:                   cfg.Host,
		CertificateAuthority:     cfg.TLSClientConfig.CAFile,
		CertificateAuthorityData: cfg.TLSClientConfig.CAData,
		InsecureSkipTLSVerify:    cfg.TLSClientConfig.Insecure,
	}
	cmdCfg.AuthInfos[DefaultCmdConfigName] = authInfo
	cmdCfg.Contexts[DefaultCmdConfigName] = &api.Context{
		Cluster:  DefaultCmdConfigName,
		AuthInfo: DefaultCmdConfigName,
	}
	cmdCfg.CurrentContext = DefaultCmdConfigName

	return clientcmd.NewDefaultClientConfig(
		*cmdCfg,
		&clientcmd.ConfigOverrides{},
	)
}

// Extracts authorization information from request header
func (self *clientManager) extractAuthInfo(req *restful.Request) (*api.AuthInfo, error) {
	if req == nil {
		log.Print("No request provided. Skipping authorization")
		return nil, nil
	}

	authHeader := req.HeaderParameter("Authorization")
	jweToken := req.HeaderParameter(JWETokenHeader)

	// Authorization header will be more important than our token
	token := self.extractTokenFromHeader(authHeader)
	if len(token) > 0 {
		return &api.AuthInfo{Token: token}, nil
	}

	if self.tokenManager != nil && len(jweToken) > 0 {
		return self.tokenManager.Decrypt(jweToken)
	}

	return nil, nil
}

func (self *clientManager) extractTokenFromHeader(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}

// Initializes client manager
func (self *clientManager) init() {
	self.initInClusterConfig()
	self.initCSRFKey()
	self.initInsecureClient()
}

// Initializes in-cluster config if apiserverHost and kubeConfigPath were not provided.
func (self *clientManager) initInClusterConfig() {
	if len(self.apiserverHost) > 0 || len(self.kubeConfigPath) > 0 {
		log.Print("Skipping in-cluster config")
		return
	}

	log.Print("Using in-cluster config to connect to apiserver")
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Could not init in cluster config: %s", err.Error())
		return
	}

	self.inClusterConfig = cfg
}

// Initializes csrfKey. If in-cluster config is detected then csrf key is initialized with
// service account token, otherwise it is generated
func (self *clientManager) initCSRFKey() {
	if self.inClusterConfig == nil {
		// Most likely running for a dev, so no replica issues, just generate a random key
		log.Println("Using random key for csrf signing")
		self.generateCSRFKey()
		return
	}

	// We run in a cluster, so we should use a signing key that is the same for potential replications
	log.Println("Using service account token for csrf signing")
	self.csrfKey = self.inClusterConfig.BearerToken
}

func (self *clientManager) initInsecureClient() {
	insecureClient, err := self.Client(nil)
	if err != nil {
		panic(err)
	}

	self.insecureClient = insecureClient
}

// Generates random csrf key
func (self *clientManager) generateCSRFKey() {
	bytes := make([]byte, 256)
	_, err := rand.Read(bytes)
	if err != nil {
		panic("Fatal error. Could not generate csrf key")
	}

	self.csrfKey = string(bytes)
}

// Returns true if in-cluster config is used
func (self *clientManager) isRunningInCluster() bool {
	return self.inClusterConfig != nil
}

// NewClientManager creates client manager based on kubeConfigPath and apiserverHost parameters.
// If both are empty then in-cluster config is used.
func NewClientManager(kubeConfigPath, apiserverHost string) clientapi.ClientManager {
	result := &clientManager{
		kubeConfigPath: kubeConfigPath,
		apiserverHost:  apiserverHost,
	}

	result.init()
	return result
}
