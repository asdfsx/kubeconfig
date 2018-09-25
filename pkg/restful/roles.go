package restful

import (
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"net/http"
	"strings"
)

type RolesResource struct {
	k8sClient               kubernetes.Interface
	selfDefineResourcePrefix string
}

func createRoleResource(k8sClient kubernetes.Interface, prefix string) (resource *RolesResource) {
	resource = &RolesResource{
		k8sClient:               k8sClient,
		selfDefineResourcePrefix: prefix,
	}
	return
}

func (rr RolesResource) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/roles").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"roles"}

	ws.Route(ws.GET("/{namespace}").To(rr.findAllRoles).
		// docs
		Doc("find all roles under specified namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]string{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{namespace}/{role}").To(rr.getRole).
		// docs
		Doc("find specified role under specified namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Param(ws.PathParameter("role", "identifier of the role").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(rbac.Role{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{namespace}/{role}").To(rr.removeRole).
		// docs
		Doc("delete specified role in specified namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string")).
		Param(ws.PathParameter("role", "identifier of the role").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes("").
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	return ws
}

// GET http://localhost:8080/namespaces
//
func (rr RolesResource) findAllRoles(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	roles, err := rr.k8sClient.RbacV1().Roles(nameOfSpace).List(metaV1.ListOptions{})
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
func (rr RolesResource) getRole(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	nameOfRole := request.PathParameter("role")
	role, err := rr.k8sClient.RbacV1().Roles(nameOfSpace).Get(nameOfRole, metaV1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(role)
}

// DELETE http://localhost:8080/namespaces/default
//
func (rr RolesResource) removeRole(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	nameOfRole := request.PathParameter("role")

	if !strings.HasPrefix(nameOfSpace, rr.selfDefineResourcePrefix) {
		response.WriteError(http.StatusBadRequest,
			errors.New(fmt.Sprintf("namespace: %s is not self define resouce, cannot remove through role!", nameOfSpace)))
		return
	}

	err := rr.k8sClient.RbacV1().Roles(nameOfRole).Delete(nameOfRole, &metaV1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.Write([]byte("{\"status\":\"success\"}"))
}
