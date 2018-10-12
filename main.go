package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/starcloud-ai/kubeconfig/pkg/restful"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
)

var (
	masterURL       string
	kubeconfig      string
	swaggerUIDist   string
	incluster       bool
	namespacePrefix = "clustar-"
	tillerRole      = "tiller-user"
	tillerNamespace = "kube-system"
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&swaggerUIDist, "swagger-ui-dist", "", "The path of the swagger-ui-dist. ")
	flag.BoolVar(&incluster, "incluster", false, "Deploy the server inside the cluster or outside the cluster")
}

func main() {
	flag.Parse()

	var cfg *rest.Config
	var err error

	var clusterServer string
	var clusterCAData []byte

	if incluster {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			glog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	clusterServer = cfg.Host
	clusterCAData = cfg.CAData

	if t := os.Getenv("NAMESPACE_PREFIX"); t != "" {
		namespacePrefix = t
	}
	if t := os.Getenv("TILLER_ROLE"); t != "" {
		tillerRole = t
	}
	if t := os.Getenv("TILLER_NAMESPACE"); t != "" {
		tillerNamespace = t
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubeclient: %s", err.Error())
	}

	handler := restful.CreateHandler(clientSet,
		namespacePrefix,
		clusterServer,
		clusterCAData,
		tillerNamespace,
		tillerRole,
		swaggerUIDist)
	err = http.ListenAndServe(":8085", handler)
	if err != nil {
		glog.Fatalf("Error running http server: %s", err.Error())
	}
}
