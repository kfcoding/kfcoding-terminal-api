package api

import (
	"github.com/emicklei/go-restful"
	authApi "github.com/terminal-controller/auth/api"
	"k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ClientManager is responsible for initializing and creating clients to communicate with
// kubernetes apiserver on demand.
type ClientManager interface {
	Client(req *restful.Request) (kubernetes.Interface, error)
	InsecureClient() kubernetes.Interface
	CanI(req *restful.Request, ssar *v1.SelfSubjectAccessReview) bool
	Config(req *restful.Request) (*rest.Config, error)
	ClientCmdConfig(req *restful.Request) (clientcmd.ClientConfig, error)
	CSRFKey() string
	HasAccess(authInfo api.AuthInfo) error
	VerberClient(req *restful.Request) (ResourceVerber, error)
	SetTokenManager(manager authApi.TokenManager)
}

// ResourceVerber is responsible for performing generic CRUD operations on all supported resources.
type ResourceVerber interface {
	Put(kind string, namespaceSet bool, namespace string, name string,
		object *runtime.Unknown) error
	Get(kind string, namespaceSet bool, namespace string, name string) (runtime.Object, error)
	Delete(kind string, namespaceSet bool, namespace string, name string) error
}

// CanIResponse is used to as response to check whether or not user is allowed to access given endpoint.
type CanIResponse struct {
	Allowed bool `json:"allowed"`
}
