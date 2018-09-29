package restful

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

func CreateHandler(k8sClient kubernetes.Interface, prefix string, clusterCAServer string, clusterCAData []byte,
	tillerNamespace string, tillerRole string, swaggerUIDist string) http.Handler {
	container := restful.NewContainer()

	nsr := createNameSpacesResource(k8sClient, prefix)
	container.Add(nsr.WebService())

	kcr := createKubeConfigResource(k8sClient,
		clusterCAServer,
		clusterCAData,
		tillerNamespace,
		tillerRole,
		prefix)
	container.Add(kcr.WebService())

	sar := createServiceAccountResource(k8sClient, prefix)
	container.Add(sar.WebService())

	rr := createRoleResource(k8sClient, prefix)
	container.Add(rr.WebService())

	crr := createClusterRoleResource(k8sClient)
	container.Add(crr.WebService())

	config := restfulspec.Config{
		WebServices:                   container.RegisteredWebServices(), // you control what services are visible
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}
	container.Add(restfulspec.NewOpenAPIService(config))

	container.Handle("/apidocs/", http.StripPrefix("/apidocs/",
		http.FileServer(http.Dir(swaggerUIDist))))
	// Optionally, you may need to enable CORS for the UI to work.
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      container}
	container.Filter(cors.Filter)

	container.ServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome!\n")
	})

	return container
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Schemes = []string{"http", "https"}
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "KubeService",
			Description: "Resource for managing Namespaces",
			Contact: &spec.ContactInfo{
				Name:  "asdfsx",
				Email: "asdfsx@gmail.com",
			},
			License: &spec.License{
				Name: "Apache 2.0",
				URL:  "http://www.apache.org/licenses/LICENSE-2.0.html",
			},
			Version: "1.0.0",
		},
	}
	swo.Tags = []spec.Tag{{TagProps: spec.TagProps{
		Name:        "Kubeconfig",
		Description: "Managing namespaces"}}}
}

type Result struct {
	Status string `json:"status" description:"action result"`
}
