package restful

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NameSpacesResource struct {
	k8sclient               kubernetes.Interface
	selfDefineResourePrefix string
}

func createNameSpacesResource(k8sclient kubernetes.Interface, prefix string) (resource *NameSpacesResource) {
	resource = &NameSpacesResource{
		k8sclient:               k8sclient,
		selfDefineResourePrefix: prefix,
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
		Writes(nil). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.PUT("/{namespace}").To(nsr.createNamespace).
		// docs
		Doc("create a namespace").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string")))

	ws.Route(ws.DELETE("/{namespace}").To(nsr.removeNamespace).
		// docs
		Doc(fmt.Sprintf("delete a namespace which prefix is %s", nsr.selfDefineResourePrefix)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string")))

	return ws
}

// GET http://localhost:8080/namespaces
//
func (nsr NameSpacesResource) findAllNamespaces(request *restful.Request, response *restful.Response) {
	namespaces, err := nsr.k8sclient.CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	list := []string{}
	for _, each := range namespaces.Items {
		list = append(list, each.Name)
	}
	response.WriteEntity(list)
}

// GET http://localhost:8080/namespaces/default
//
func (nsr NameSpacesResource) findNamespace(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	namespace, err := nsr.k8sclient.CoreV1().Namespaces().Get(nameofspace, meta_v1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(namespace)
}

// PUT http://localhost:8080/namespaces/clustar-{name}
//
func (nsr *NameSpacesResource) createNamespace(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	if !strings.HasPrefix(nameofspace, nsr.selfDefineResourePrefix) {
		nameofspace = fmt.Sprintf("%s-%s", nsr.selfDefineResourePrefix, nameofspace)
	}
	namespacetmp := &core_v1.Namespace{}
	namespacetmp.APIVersion = "v1"
	namespacetmp.Kind = "Namespace"
	namespacetmp.Name = nameofspace
	namespacetmp, err := nsr.k8sclient.CoreV1().Namespaces().Create(namespacetmp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		response.WriteEntity(namespacetmp)
	}
}

// DELETE http://localhost:8080/namespaces/clustar-{name}
//
func (nsr *NameSpacesResource) removeNamespace(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	if strings.HasPrefix(nameofspace, nsr.selfDefineResourePrefix) {
		err := nsr.k8sclient.CoreV1().Namespaces().Delete(nameofspace, &meta_v1.DeleteOptions{})
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
		response.Write([]byte("{\"status\":\"success\"}"))
	} else {
		response.WriteError(http.StatusBadRequest,
			errors.New(fmt.Sprintf("namespace: %s is not self define resouce, cannot remove through service!")))
	}
}
