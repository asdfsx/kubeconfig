package restful

import (
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/ghodss/yaml"
	jsonitor "github.com/json-iterator/go"
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
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

	// check if cluster role is exists
	// if not exists then create it
	statenum, err = kcr.checkClusterRole(action)
	if err != nil {
		return statenum, errors.New(fmt.Sprintf("error while checkClusterRole:%s", err))
	}

	// check if cluster rolebinding is exists
	// if not exists then create it
	statenum, err = kcr.checkClusterRoleBinding(action)
	if err != nil {
		return statenum, errors.New(fmt.Sprintf("error while checkClusterRoleBinding:%s", err))
	}

	// check if role is exists
	// if not exists then create it
	statenum, err = kcr.checkRole(action)
	if err != nil {
		return statenum, errors.New(fmt.Sprintf("error while checkRole:%s", err))
	}

	// check if rolebinding is exists
	// if not exists then create it
	statenum, err = kcr.checkRoleBinding(action)
	if err != nil {
		return statenum, errors.New(fmt.Sprintf("error while checkRoleBinding:%s", err))
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

// tiller 的 role 应该在安装helm的时候创建好，所以这里只是检查是否存在，不执行创建了
func (kcr KubeConfigResource) checkRole(action *serviceAccountAction) (int, error) {
	_, err := kcr.k8sClient.RbacV1().Roles(kcr.tillerNamespace).Get(kcr.tillerRole, metaV1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	} else {
		return http.StatusInternalServerError, err
	}
}

func (kcr KubeConfigResource) checkClusterRole(action *serviceAccountAction) (int, error) {
	_, err := kcr.k8sClient.RbacV1().ClusterRoles().Get(readOnlyRole, metaV1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8sError.StatusError:
		if t.Status().Reason == metaV1.StatusReasonNotFound {
			roletmp := &rbacV1.ClusterRole{}
			roletmp.APIVersion = "v1"
			roletmp.Kind = "ClusterRole"
			roletmp.Name = readOnlyRole
			roletmp.Namespace = action.NameSpace
			roletmp.Rules = append(roletmp.Rules, rbacV1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			})
			_, err := kcr.k8sClient.RbacV1().ClusterRoles().Create(roletmp)
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
	bindingName := getRoleBindingName(action.NameSpace, action.ServiceAccount, readOnlyRole)
	_, err := kcr.k8sClient.RbacV1().ClusterRoleBindings().Get(bindingName, metaV1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8sError.StatusError:
		if t.Status().Reason == metaV1.StatusReasonNotFound {
			rolebindingtmp := &rbacV1.ClusterRoleBinding{}
			rolebindingtmp.APIVersion = "v1"
			rolebindingtmp.Kind = "ClusterRoleBinding"
			rolebindingtmp.Name = bindingName
			rolebindingtmp.Namespace = action.NameSpace
			rolebindingtmp.Subjects = append(rolebindingtmp.Subjects, rbacV1.Subject{
				Kind:      "ServiceAccount",
				Name:      action.ServiceAccount,
				Namespace: action.NameSpace,
			})
			rolebindingtmp.RoleRef.Kind = "ClusterRole"
			rolebindingtmp.RoleRef.Name = readOnlyRole

			_, err = kcr.k8sClient.RbacV1().ClusterRoleBindings().Create(rolebindingtmp)
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}
	return http.StatusOK, nil
}

func (kcr KubeConfigResource) checkRoleBinding(action *serviceAccountAction) (int, error) {
	bindingName := getRoleBindingName(kcr.tillerNamespace, action.ServiceAccount, kcr.tillerRole)
	_, err := kcr.k8sClient.RbacV1().RoleBindings(kcr.tillerNamespace).Get(bindingName, metaV1.GetOptions{})
	if err == nil {
		return http.StatusOK, nil
	}

	switch t := err.(type) {
	case *k8sError.StatusError:
		if t.Status().Reason == metaV1.StatusReasonNotFound {
			rolebindingtmp := &rbacV1.RoleBinding{}
			rolebindingtmp.APIVersion = "v1"
			rolebindingtmp.Kind = "RoleBinding"
			rolebindingtmp.Name = bindingName
			rolebindingtmp.Namespace = kcr.tillerNamespace
			rolebindingtmp.Subjects = append(rolebindingtmp.Subjects, rbacV1.Subject{
				Kind:      "ServiceAccount",
				Name:      action.ServiceAccount,
				Namespace: action.NameSpace,
			})
			rolebindingtmp.RoleRef.Kind = "Role"
			rolebindingtmp.RoleRef.Name = kcr.tillerRole

			_, err = kcr.k8sClient.RbacV1().RoleBindings(kcr.tillerNamespace).Create(rolebindingtmp)
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
	nameOfSpace := request.PathParameter("namespace")
	nameOfAccount := request.PathParameter("serviceAccount")

	if !strings.HasPrefix(nameOfSpace, kcr.selfDefineResourcePrefix) {
		response.WriteError(http.StatusBadRequest, errors.New(
			fmt.Sprintf("namespace: %s is not self define resouce, cannot remove through service!", nameOfSpace)))
		return
	}

	bindingName := getRoleBindingName(nameOfSpace, nameOfAccount, readOnlyRole)
	err := kcr.k8sClient.RbacV1().ClusterRoleBindings().Delete(bindingName, &metaV1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	bindingName = getRoleBindingName(kcr.tillerNamespace, nameOfAccount, kcr.tillerRole)
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
