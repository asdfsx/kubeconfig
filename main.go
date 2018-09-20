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

	"github.com/asdfsx/kubeconfig/pkg/restful"
)

var (
	masterURL     string
	kubeconfig    string
	swaggerUIDist string
	incluster     bool
)

const GLOBAL_PREFIX = "clustar"

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&swaggerUIDist, "swagger-ui-dist", "", "The path of the swagger-ui-dist. ")
	flag.BoolVar(&incluster, "incluster", false, "Deploy the server inside the cluster or outside the cluster")
}

type Item struct {
	Seq    int
	Result map[string]int
}

type Message struct {
	Dept    string
	Subject string
	Time    int64
	Detail  []Item
}

type kubeClient struct {
	k8sclient       kubernetes.Interface
	cluster_server  string
	cluster_ca_data []byte
}

func main() {
	flag.Parse()

	var cfg *rest.Config
	var err error

	var cluster_server string
	var cluster_ca_data []byte

	if incluster {
		cfg, err = rest.InClusterConfig()
		cluster_server = os.Getenv("CLUSTER_SERVER")
		tmp := os.Getenv("CLUSTER_CA_DATA")
		cluster_ca_data, err = base64.StdEncoding.DecodeString(tmp)
		if err != nil {
			glog.Fatalf("Error decoding ca-data: %s", err.Error())
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		cluster_server = cfg.Host
		cluster_ca_data = cfg.CAData
	}

	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubeclient: %s", err.Error())
	}

	handler := restful.CreateHandler(clientset, GLOBAL_PREFIX, cluster_server, cluster_ca_data, swaggerUIDist)
	http.ListenAndServe(":8085", handler)
}
