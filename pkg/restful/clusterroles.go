package restful

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"net/http"
)

type ClusterRoleResource struct {
	k8sClient               kubernetes.Interface
}

func createClusterRoleResource(k8sClient kubernetes.Interface) (resource *ClusterRoleResource) {
	resource = &ClusterRoleResource{
		k8sClient:               k8sClient,
	}
	return
}

func (crr ClusterRoleResource) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/clusterroles").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"clusterroles"}

	ws.Route(ws.GET("/").To(crr.findAllClusterRoles).
		// docs
		Doc("find all cluster roles").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]string{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{clusterRole}").To(crr.getClusterRole).
		// docs
		Doc("find specified cluster role").
		Param(ws.PathParameter("clusterRole", "identifier of the role").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(rbac.ClusterRole{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	return ws
}

// GET http://localhost:8080/namespaces
//
func (crr ClusterRoleResource) findAllClusterRoles(request *restful.Request, response *restful.Response) {
	roles, err := crr.k8sClient.RbacV1().ClusterRoles().List(metaV1.ListOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	var list []string
	for _, each := range roles.Items {
		list = append(list, each.Name)
	}
	response.WriteEntity(list)
}

// GET http://localhost:8080/namespaces/default
//
func (crr ClusterRoleResource) getClusterRole(request *restful.Request, response *restful.Response) {
	nameOfRole := request.PathParameter("clusterRole")
	role, err := crr.k8sClient.RbacV1().ClusterRoles().Get(nameOfRole, metaV1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(role)
}
