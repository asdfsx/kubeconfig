package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
)

var (
	masterURL  string
	kubeconfig string
	incluster  bool
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
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
	k8sclient kubernetes.Interface
	cluster_server string
	cluster_ca_data []byte
}

func (kc kubeClient) kubeconfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	namespace := ps.ByName("namespace")
	accountName := ps.ByName("serviceAccount")

	serviceAccount, err := kc.k8sclient.CoreV1().ServiceAccounts(namespace).Get(accountName, meta_v1.GetOptions{})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	secret, err := kc.k8sclient.CoreV1().Secrets(namespace).Get(serviceAccount.Secrets[0].Name, meta_v1.GetOptions{})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}
	config := generateConfigMap2(serviceAccount.Name, secret.Data["token"], kc.cluster_server, kc.cluster_ca_data)
	result, err := json.Marshal(config)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}

	output, err := yaml.JSONToYAML(result)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, err.Error())
		return
	}
	fmt.Fprintf(w, "%s", output)
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
		if err != nil{
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

	kc := kubeClient{
		clientset,
		cluster_server,
		cluster_ca_data,
	}

	router := httprouter.New()
	router.GET("/kubeconfig/:namespace/:serviceAccount", kc.kubeconfig)
	router.HandlerFunc("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome!\n")
	})
	http.ListenAndServe(":8085", router)
}
