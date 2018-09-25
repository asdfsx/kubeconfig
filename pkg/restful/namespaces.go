package restful

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NameSpacesResource struct {
	k8sClient               kubernetes.Interface
	selfDefineResourcePrefix string
}

func createNameSpacesResource(k8sclient kubernetes.Interface, prefix string) (resource *NameSpacesResource) {
	resource = &NameSpacesResource{
		k8sClient:               k8sclient,
		selfDefineResourcePrefix: prefix,
	}
	return
}

func (nsr NameSpacesResource) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/namespaces").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"namespaces"}

	ws.Route(ws.GET("/").To(nsr.findAllNamespaces).
		// docs
		Doc("get all namespaces").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]string{}).
		Returns(200, "OK", []string{}))

	ws.Route(ws.GET("/{namespace}").To(nsr.findNamespace).
		// docs
		Doc("get a namespace by name").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(coreV1.Namespace{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.PUT("/{namespace}").To(nsr.createNamespace).
		// docs
		Doc("create a namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(serviceAccountAction{}).
		Writes(coreV1.Namespace{}).
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{namespace}").To(nsr.removeNamespace).
		// docs
		Doc(fmt.Sprintf("delete a namespace which prefix is %s", nsr.selfDefineResourcePrefix)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string")).
		Writes("").
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	return ws
}

// GET http://localhost:8080/namespaces
//
func (nsr NameSpacesResource) findAllNamespaces(request *restful.Request, response *restful.Response) {
	namespaces, err := nsr.k8sClient.CoreV1().Namespaces().List(metaV1.ListOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	var list []string
	for _, each := range namespaces.Items {
		list = append(list, each.Name)
	}
	response.WriteEntity(list)
}

// GET http://localhost:8080/namespaces/default
//
func (nsr NameSpacesResource) findNamespace(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	namespace, err := nsr.k8sClient.CoreV1().Namespaces().Get(nameOfSpace, metaV1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(namespace)
}

// PUT http://localhost:8080/namespaces/clustar-{name}
//
func (nsr *NameSpacesResource) createNamespace(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	if !strings.HasPrefix(nameOfSpace, nsr.selfDefineResourcePrefix) {
		response.WriteError(http.StatusBadRequest, errors.New(
			fmt.Sprintf("namespace: %s is not self define resouce, cannot use through service!", nameOfSpace)))
		return
	}
	namespaceTmp := &coreV1.Namespace{}
	namespaceTmp.APIVersion = "v1"
	namespaceTmp.Kind = "Namespace"
	namespaceTmp.Name = nameOfSpace
	namespaceTmp, err := nsr.k8sClient.CoreV1().Namespaces().Create(namespaceTmp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		response.WriteEntity(namespaceTmp)
	}
}

// DELETE http://localhost:8080/namespaces/clustar-{name}
//
func (nsr *NameSpacesResource) removeNamespace(request *restful.Request, response *restful.Response) {
	nameOfSpace := request.PathParameter("namespace")
	if strings.HasPrefix(nameOfSpace, nsr.selfDefineResourcePrefix) {
		err := nsr.k8sClient.CoreV1().Namespaces().Delete(nameOfSpace, &metaV1.DeleteOptions{})
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
		response.Write([]byte("{\"status\":\"success\"}"))
	} else {
		response.WriteError(http.StatusBadRequest,
			errors.New(fmt.Sprintf("namespace: %s is not self define resouce, cannot remove through service!", nameOfSpace)))
	}
}
