package main

import (
	"encoding/base64"
	"flag"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"

	"github.com/starcloud-ai/kubeconfig/pkg/restful"
)

var (
	masterURL     string
	kubeconfig    string
	swaggerUIDist string
	incluster     bool
)

const globalPrefix = "clustar-"

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
		tmp := os.Getenv("CLUSTER_CA_DATA")
		clusterCAData, err = base64.StdEncoding.DecodeString(tmp)
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

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubeclient: %s", err.Error())
	}

	handler := restful.CreateHandler(clientSet, globalPrefix, clusterServer, clusterCAData, swaggerUIDist)
	http.ListenAndServe(":8085", handler)
}
