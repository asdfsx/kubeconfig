package restful

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/ghodss/yaml"
	jsonitor "github.com/json-iterator/go"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8s_cli_api "k8s.io/client-go/tools/clientcmd/api/v1"
	"net/http"
)

type KubeConfigResource struct {
	k8sclient     kubernetes.Interface
	clusterServer string
	clusterCAData []byte
}

func createKubeConfigResource(k8sclient kubernetes.Interface, clusterServer string, clusterCAData []byte) (resource *KubeConfigResource) {
	resource = &KubeConfigResource{
		k8sclient:     k8sclient,
		clusterServer: clusterServer,
		clusterCAData: clusterCAData,
	}
	return
}

func (kcr KubeConfigResource) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/kubeconfig").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"kubeconfig"}

	ws.Route(ws.GET("/{namespace}/{serviceAccount}").To(kcr.generateKubeConfig).
		// docs
		Doc("generate kubeconfig for specified serviceAccount").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Param(ws.PathParameter("serviceAccount", "identifier of the serviceAccount").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(nil). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	return ws
}

// GET http://localhost:8080/kubeconfig/default/default
//
func (kcr KubeConfigResource) generateKubeConfig(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	nameofaccount := request.PathParameter("namespace")

	serviceAccount, err := kcr.k8sclient.CoreV1().ServiceAccounts(nameofspace).Get(nameofaccount, meta_v1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	secret, err := kcr.k8sclient.CoreV1().Secrets(nameofspace).Get(serviceAccount.Secrets[0].Name, meta_v1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	config := generateConfigMap2(serviceAccount.Name, secret.Data["token"], kcr.clusterServer, kcr.clusterCAData)
	result, err := jsonitor.Marshal(config)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	output, err := yaml.JSONToYAML(result)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.Write(output)
}

func generateConfigMap2(name string, token []byte, server string, caData []byte) (confMap *k8s_cli_api.Config) {
	confMap = &k8s_cli_api.Config{}
	confMap.APIVersion = "v1"
	confMap.Kind = "Config"
	confMap.CurrentContext = name
	confMap.Contexts = append(confMap.Contexts, k8s_cli_api.NamedContext{
		Name: name,
		Context: k8s_cli_api.Context{
			AuthInfo: name,
			Cluster:  name,
		},
	})
	confMap.AuthInfos = append(confMap.AuthInfos, k8s_cli_api.NamedAuthInfo{
		Name: name,
		AuthInfo: k8s_cli_api.AuthInfo{
			Token: fmt.Sprintf("%s", token),
		},
	})
	confMap.Clusters = append(confMap.Clusters, k8s_cli_api.NamedCluster{
		Name: name,
		Cluster: k8s_cli_api.Cluster{
			Server:                   server,
			CertificateAuthorityData: caData,
		},
	})
	return
}
