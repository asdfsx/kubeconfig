package main

import (
	"encoding/base64"
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
		clusterServer = os.Getenv("CLUSTER_SERVER")
		clusterCADataOriginal := os.Getenv("CLUSTER_CA_DATA")

		clusterCAData, err = base64.StdEncoding.DecodeString(clusterCADataOriginal)
		if err != nil {
			glog.Fatalf("Error decoding ca-data: %s", err.Error())
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			glog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
		clusterServer = cfg.Host
		clusterCAData = cfg.CAData
	}

	if t := os.Getenv("NAMESPACE_PREFIX"); t != "" {
		namespacePrefix = t
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubeclient: %s", err.Error())
	}

	handler := restful.CreateHandler(clientSet, namespacePrefix, clusterServer, clusterCAData, swaggerUIDist)
	err = http.ListenAndServe(":8085", handler)
	if err != nil {
		glog.Fatalf("Error running http server: %s", err.Error())
	}
}
