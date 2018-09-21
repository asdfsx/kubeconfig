package restful

import (
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/ghodss/yaml"
	jsonitor "github.com/json-iterator/go"
	core_v1 "k8s.io/api/core/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	k8s_error "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8s_cli_api "k8s.io/client-go/tools/clientcmd/api/v1"
	"net/http"
	"strings"
)

const READONLYROLE = "cluster-readonly"
const ROLEBINDINGNAME = "%s:%s:%s-binding"

func getRoleBindingName(namespace, serviceaccount string) string {
	return fmt.Sprintf(ROLEBINDINGNAME, namespace, serviceaccount, READONLYROLE)
}

type KubeConfigResource struct {
	k8sclient               kubernetes.Interface
	selfDefineResourePrefix string
	clusterServer           string
	clusterCAData           []byte
}

func createKubeConfigResource(k8sclient kubernetes.Interface, clusterServer string, clusterCAData []byte, prefix string) (resource *KubeConfigResource) {
	resource = &KubeConfigResource{
		k8sclient:               k8sclient,
		clusterServer:           clusterServer,
		clusterCAData:           clusterCAData,
		selfDefineResourePrefix: prefix,
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
		Writes(k8s_cli_api.Config{}). // on the response
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
	nameofspace := request.PathParameter("namespace")
	nameofaccount := request.PathParameter("serviceAccount")

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
		return statenum, err
	}

	// check if serviceAccount is exsists
	// if not exists then create it
	statenum, err = kcr.checkServiceAccount(action)
	if err != nil {
		return statenum, err
	}

	// check if role is exists
	// if not exists then create it
	statenum, err = kcr.checkClusterRole(action)
	if err != nil {
		return statenum, err
	}

	// check if rolebinding is exists
	// if not exists then create it
	statenum, err = kcr.checkClusterRoleBinding(action)
	if err != nil {
		return statenum, err
	}

	return http.StatusOK, nil
}

func (kcr KubeConfigResource) checkNamespace(action *serviceAccountAction) (int, error) {
	if !strings.HasPrefix(action.NameSpace, kcr.selfDefineResourePrefix) {
		return http.StatusBadRequest, errors.New(
			fmt.Sprintf("namespace: %s is not self define resouce, cannot use through service!", action.NameSpace))
	}

	_, err := kcr.k8sclient.CoreV1().Namespaces().Get(action.NameSpace, meta_v1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8s_error.StatusError:
		if t.Status().Reason == meta_v1.StatusReasonNotFound {
			namespacetmp := &core_v1.Namespace{}
			namespacetmp.APIVersion = "v1"
			namespacetmp.Kind = "Namespace"
			namespacetmp.Name = action.NameSpace
			_, err = kcr.k8sclient.CoreV1().Namespaces().Create(namespacetmp)
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
	_, err := kcr.k8sclient.CoreV1().ServiceAccounts(action.NameSpace).Get(action.ServiceAccount, meta_v1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8s_error.StatusError:
		if t.Status().Reason == meta_v1.StatusReasonNotFound {
			serviceaccounttmp := &core_v1.ServiceAccount{}
			serviceaccounttmp.APIVersion = "v1"
			serviceaccounttmp.Kind = "ServiceAccount"
			serviceaccounttmp.Name = action.ServiceAccount
			serviceaccounttmp.Namespace = action.NameSpace
			_, err = kcr.k8sclient.CoreV1().ServiceAccounts(action.NameSpace).Create(serviceaccounttmp)
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}
	return http.StatusOK, nil
}

func (kcr KubeConfigResource) checkClusterRole(action *serviceAccountAction) (int, error) {
	_, err := kcr.k8sclient.RbacV1().ClusterRoles().Get(READONLYROLE, meta_v1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8s_error.StatusError:
		if t.Status().Reason == meta_v1.StatusReasonNotFound {
			roletmp := &rbac_v1.ClusterRole{}
			roletmp.APIVersion = "v1"
			roletmp.Kind = "ClusterRole"
			roletmp.Name = READONLYROLE
			roletmp.Namespace = action.NameSpace
			roletmp.Rules = append(roletmp.Rules, rbac_v1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			})
			_, err := kcr.k8sclient.RbacV1().ClusterRoles().Create(roletmp)
			fmt.Println(err)
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}
	return http.StatusOK, nil
}

func (kcr KubeConfigResource) checkClusterRoleBinding(action *serviceAccountAction) (int, error) {
	bindingName := getRoleBindingName(action.NameSpace, action.ServiceAccount)
	_, err := kcr.k8sclient.RbacV1().ClusterRoleBindings().Get(bindingName, meta_v1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8s_error.StatusError:
		if t.Status().Reason == meta_v1.StatusReasonNotFound {
			rolebindingtmp := &rbac_v1.ClusterRoleBinding{}
			rolebindingtmp.APIVersion = "v1"
			rolebindingtmp.Kind = "ClusterRoleBinding"
			rolebindingtmp.Name = bindingName
			rolebindingtmp.Namespace = action.NameSpace
			rolebindingtmp.Subjects = append(rolebindingtmp.Subjects, rbac_v1.Subject{
				Kind:      "ServiceAccount",
				Name:      action.ServiceAccount,
				Namespace: action.NameSpace,
			})
			rolebindingtmp.RoleRef.Kind = "ClusterRole"
			rolebindingtmp.RoleRef.Name = READONLYROLE

			_, err = kcr.k8sclient.RbacV1().ClusterRoleBindings().Create(rolebindingtmp)
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}
	return http.StatusOK, nil
}

func (kcr KubeConfigResource) deleteServiceAccount(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	nameofaccount := request.PathParameter("serviceAccount")

	if !strings.HasPrefix(nameofspace, kcr.selfDefineResourePrefix) {
		response.WriteError(http.StatusBadRequest, errors.New(
			fmt.Sprintf("namespace: %s is not self define resouce, cannot remove through service!", nameofspace)))
		return
	}

	bindingName := getRoleBindingName(nameofspace, nameofaccount)
	err := kcr.k8sclient.RbacV1().ClusterRoleBindings().Delete(bindingName, &meta_v1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	err = kcr.k8sclient.CoreV1().ServiceAccounts(nameofspace).Delete(nameofaccount, &meta_v1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.Write([]byte("{\"status\":\"success\"}"))
}

func generateConfigMap(name string, token []byte, server string, caData []byte) (confMap *k8s_cli_api.Config) {
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

//
type serviceAccountAction struct {
	NameSpace      string `json:"namespace" description:"name of the namespace"`
	ServiceAccount string `json:"serviceaccount" description:"name of the service account"`
	//Role           string  `json:"role,omitempty" description:"role bind to service account"`
	//ClusterRole    string  `json:"clusterole,omitempty" description:"cluster role bind to service account"`
}
