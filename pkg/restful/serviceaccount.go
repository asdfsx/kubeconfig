package restful

import (
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"strings"
)

type ServiceAccountResource struct {
	k8sclient               kubernetes.Interface
	selfDefineResourePrefix string
}

func createServiceAccountResource(k8sclient kubernetes.Interface, prefix string) (resource *ServiceAccountResource) {
	resource = &ServiceAccountResource{
		k8sclient:               k8sclient,
		selfDefineResourePrefix: prefix,
	}
	return
}

func (sar ServiceAccountResource) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/serviceAccount").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"ServiceAccount"}

	ws.Route(ws.GET("/{namespace}/").To(sar.findAllServiceAccount).
		// docs
		Doc("find all service account under specified namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]string{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{namespace}/{serviceAccount}").To(sar.getServiceAccount).
		// docs
		Doc("find specified service account under specified namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Param(ws.PathParameter("serviceAccount", "identifier of the serviceAccount").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(core_v1.ServiceAccount{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.POST("/{namespace}/{serviceAccount}").To(sar.createServiceAccount).
		// docs
		Doc("create service account in specified namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Param(ws.PathParameter("serviceAccount", "identifier of the serviceAccount").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(core_v1.ServiceAccount{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{namespace}/{serviceAccount}").To(sar.deleteServiceAccount).
		// docs
		Doc("create service account in specified namespace").
		Param(ws.PathParameter("namespace", "identifier of the namespace").DataType("string").DefaultValue("default")).
		Param(ws.PathParameter("serviceAccount", "identifier of the serviceAccount").DataType("string").DefaultValue("default")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(""). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	return ws
}

// GET http://localhost:8080/serviceAccount/default
//
func (sar ServiceAccountResource) findAllServiceAccount(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")

	serviceAccounts, err := sar.k8sclient.CoreV1().ServiceAccounts(nameofspace).List(meta_v1.ListOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	list := []string{}
	for _, each := range serviceAccounts.Items {
		list = append(list, each.Name)
	}
	response.WriteEntity(list)
}

// GET http://localhost:8080/serviceAccount/default/default
//
func (sar ServiceAccountResource) getServiceAccount(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	nameofaccount := request.PathParameter("serviceAccount")

	serviceAccount, err := sar.k8sclient.CoreV1().ServiceAccounts(nameofspace).Get(nameofaccount, meta_v1.GetOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(serviceAccount)
}

// PUT http://localhost:8080/serviceAccount/clustar-{ns}/default
//
func (sar ServiceAccountResource) createServiceAccount(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	nameofaccount := request.PathParameter("serviceAccount")

	if !strings.HasPrefix(nameofspace, sar.selfDefineResourePrefix) {
		response.WriteError(http.StatusBadRequest,
			errors.New(fmt.Sprintf("namespace: %s is not self define namespace, cannot create service account!")))
		return
	}

	serviceAccountTmp := &core_v1.ServiceAccount{}
	serviceAccountTmp.APIVersion = "v1"
	serviceAccountTmp.Kind = "Namespace"
	serviceAccountTmp.Namespace = nameofspace
	serviceAccountTmp.Name = nameofaccount
	serviceAccount, err := sar.k8sclient.CoreV1().ServiceAccounts(nameofspace).Create(serviceAccountTmp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(serviceAccount)
}

// DELETE http://localhost:8080/serviceAccount/clustar-{ns}/default
//
func (sar *ServiceAccountResource) deleteServiceAccount(request *restful.Request, response *restful.Response) {
	nameofspace := request.PathParameter("namespace")
	nameofaccount := request.PathParameter("serviceAccount")

	if !strings.HasPrefix(nameofspace, sar.selfDefineResourePrefix) {
		response.WriteError(http.StatusBadRequest,
			errors.New(fmt.Sprintf("namespace: %s is not self define resouce, cannot remove through service!", nameofspace)))
		return
	}

	err := sar.k8sclient.CoreV1().ServiceAccounts(nameofspace).Delete(nameofaccount, &meta_v1.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.Write([]byte("{\"status\":\"success\"}"))
}
