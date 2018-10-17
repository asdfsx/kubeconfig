package restful

import (
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/ghodss/yaml"
	jsonitor "github.com/json-iterator/go"
	"github.com/starcloud-ai/kubeconfig/pkg/rbac"
	coreV1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sCliApi "k8s.io/client-go/tools/clientcmd/api/v1"
	"net/http"
	"strings"
)

type KubeConfigResource struct {
	k8sClient                kubernetes.Interface
	clusterServer            string
	clusterCAData            []byte
	tillerNamespace          string
	tillerRole               string
	selfDefineResourcePrefix string
}

func createKubeConfigResource(k8sClient kubernetes.Interface,
	clusterServer string,
	clusterCAData []byte,
	tillerNamespace string,
	tillerRole string,
	prefix string) (resource *KubeConfigResource) {
	resource = &KubeConfigResource{
		k8sClient:               k8sClient,
		clusterServer:           clusterServer,
		clusterCAData:           clusterCAData,
		tillerNamespace:         tillerNamespace,
		tillerRole:              tillerRole,
		selfDefineResourcePrefix: prefix,
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
		Writes(k8sCliApi.Config{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.POST("/").To(kcr.createServiceAccount).
		// docs
		Doc("create serviceAccount").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(serviceAccountAction{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{namespace}/{serviceAccount}").To(kcr.deleteServiceAccount).
		// docs
		Doc("deletet serviceAccount").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Param(ws.PathParameter("serviceAccount", "identifier of the serviceAccount").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	return ws
}

// GET http://localhost:8080/kubeconfig/default/default
//
func (kcr KubeConfigResource) generateKubeConfig(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	nameOfAccount := request.PathParameter("serviceAccount")

	serviceAccount, err := kcr.k8sClient.CoreV1().ServiceAccounts(nameOfSpace).Get(nameOfAccount, metaV1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	secret, err := kcr.k8sClient.CoreV1().Secrets(nameOfSpace).Get(serviceAccount.Secrets[0].Name, metaV1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	config := generateConfigMap(serviceAccount.Name, secret.Data["token"], kcr.clusterServer, kcr.clusterCAData)
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

func (kcr KubeConfigResource) createServiceAccount(request *restful.Request, response *restful.Response) {
	serviceAccountAction := &serviceAccountAction{}
	err := request.ReadEntity(serviceAccountAction)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	statenum, err := kcr.createServiceAccountAction(serviceAccountAction)
	if err != nil {
		response.WriteError(statenum, err)
		return
	}
	response.Write([]byte("{\"status\":\"success\"}"))
	return
}

func (kcr KubeConfigResource) createServiceAccountAction(action *serviceAccountAction) (int, error) {
	// check if namespace is exists
	// if not exists then create it
	statenum, err := kcr.checkNamespace(action)
	if err != nil {
		return statenum, errors.New(fmt.Sprintf("error while checkNamespace:%s", err))
	}

	// check if serviceAccount is exsists
	// if not exists then create it
	statenum, err = kcr.checkServiceAccount(action)
	if err != nil {
		return statenum, errors.New(fmt.Sprintf("error while checkServiceAccount:%s", err))
	}

	// check if roles and bindings is exists
	// if not exists then create them
	statenum, err = kcr.checkRoleAndBinding(action)
	if err != nil {
		return statenum, errors.New(fmt.Sprintf("error while checkClusterRole:%s", err))
	}

	return http.StatusOK, nil
}

func (kcr KubeConfigResource) checkNamespace(action *serviceAccountAction) (int, error) {
	if !strings.HasPrefix(action.NameSpace, kcr.selfDefineResourcePrefix) {
		return http.StatusBadRequest, errors.New(
			fmt.Sprintf("namespace: %s is not self define resouce, cannot use through service!", action.NameSpace))
	}

	_, err := kcr.k8sClient.CoreV1().Namespaces().Get(action.NameSpace, metaV1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8sError.StatusError:
		if t.Status().Reason == metaV1.StatusReasonNotFound {
			namespacetmp := &coreV1.Namespace{}
			namespacetmp.APIVersion = "v1"
			namespacetmp.Kind = "Namespace"
			namespacetmp.Name = action.NameSpace
			_, err = kcr.k8sClient.CoreV1().Namespaces().Create(namespacetmp)
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}
	return http.StatusOK, nil
}

func (kcr KubeConfigResource) checkServiceAccount(action *serviceAccountAction) (int, error) {
	_, err := kcr.k8sClient.CoreV1().ServiceAccounts(action.NameSpace).Get(action.ServiceAccount, metaV1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8sError.StatusError:
		if t.Status().Reason == metaV1.StatusReasonNotFound {
			serviceaccounttmp := &coreV1.ServiceAccount{}
			serviceaccounttmp.APIVersion = "v1"
			serviceaccounttmp.Kind = "ServiceAccount"
			serviceaccounttmp.Name = action.ServiceAccount
			serviceaccounttmp.Namespace = action.NameSpace
			_, err = kcr.k8sClient.CoreV1().ServiceAccounts(action.NameSpace).Create(serviceaccounttmp)
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}
	return http.StatusOK, nil
}

func (kcr KubeConfigResource) checkRoleAndBinding(action *serviceAccountAction) (int, error) {
	tillerRole := rbac.NewTillerRole(kcr.tillerNamespace, kcr.tillerRole, kcr.k8sClient)
	adminRole := rbac.NewNamespaceAdminRole(action.NameSpace, kcr.k8sClient)
	clusterReadonlyRole := rbac.NewClusterReadonlyRoleRole(action.NameSpace, rbac.ReadOnlyRole, kcr.k8sClient)

	err := tillerRole.CreateRole()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = adminRole.CreateRole()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = clusterReadonlyRole.CreateRole()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = tillerRole.CreateRoleBinding(action.NameSpace, action.ServiceAccount)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = adminRole.CreateRoleBinding(action.NameSpace, action.ServiceAccount)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = clusterReadonlyRole.CreateRoleBinding(action.NameSpace, action.ServiceAccount)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (kcr KubeConfigResource) deleteServiceAccount(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	nameOfAccount := request.PathParameter("serviceAccount")

	if !strings.HasPrefix(nameOfSpace, kcr.selfDefineResourcePrefix) {
		response.WriteError(http.StatusBadRequest, errors.New(
			fmt.Sprintf("namespace: %s is not self define resouce, cannot remove through service!", nameOfSpace)))
		return
	}

	bindingName := rbac.GenerateRoleBindingName(rbac.ReadOnlyRole, nameOfSpace, nameOfAccount)
	err := kcr.k8sClient.RbacV1().ClusterRoleBindings().Delete(bindingName, &metaV1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	bindingName = rbac.GenerateRoleBindingName(kcr.tillerRole, kcr.tillerNamespace, nameOfAccount)
	err = kcr.k8sClient.RbacV1().RoleBindings(kcr.tillerNamespace).Delete(bindingName, &metaV1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	err = kcr.k8sClient.CoreV1().ServiceAccounts(nameOfSpace).Delete(nameOfAccount, &metaV1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.Write([]byte("{\"status\":\"success\"}"))
}

func generateConfigMap(name string, token []byte, server string, caData []byte) (confMap *k8sCliApi.Config) {
	confMap = &k8sCliApi.Config{}
	confMap.APIVersion = "v1"
	confMap.Kind = "Config"
	confMap.CurrentContext = name
	confMap.Contexts = append(confMap.Contexts, k8sCliApi.NamedContext{
		Name: name,
		Context: k8sCliApi.Context{
			AuthInfo: name,
			Cluster:  name,
		},
	})
	confMap.AuthInfos = append(confMap.AuthInfos, k8sCliApi.NamedAuthInfo{
		Name: name,
		AuthInfo: k8sCliApi.AuthInfo{
			Token: fmt.Sprintf("%s", token),
		},
	})
	confMap.Clusters = append(confMap.Clusters, k8sCliApi.NamedCluster{
		Name: name,
		Cluster: k8sCliApi.Cluster{
			Server:                   server,
			CertificateAuthorityData: caData,
		},
	})
	return
}

//
type serviceAccountAction struct {
	NameSpace      string `json:"namespace" description:"name of the namespace"`
	ServiceAccount string `json:"serviceaccount" description:"name of the service account"`
	//Role           string  `json:"role,omitempty" description:"role bind to service account"`
	//ClusterRole    string  `json:"clusterole,omitempty" description:"cluster role bind to service account"`
}
