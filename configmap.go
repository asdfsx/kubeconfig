package main

import (
	"fmt"
	k8s_cli_api "k8s.io/client-go/tools/clientcmd/api/v1"
)

func generateConfigMap2(name string, token []byte, server string, caData []byte) (confMap *k8s_cli_api.Config) {
	confMap = &k8s_cli_api.Config{}
	confMap.APIVersion = "v1"
	confMap.Kind = "Config"
	confMap.CurrentContext = name
	confMap.Contexts = append(confMap.Contexts, k8s_cli_api.NamedContext{
		Name: name,
		Context: k8s_cli_api.Context{
			AuthInfo: name,
			Cluster: name,
		},
	})
	confMap.AuthInfos = append(confMap.AuthInfos, k8s_cli_api.NamedAuthInfo{
		Name: name,
		AuthInfo: k8s_cli_api.AuthInfo{
			Token:    fmt.Sprintf("%s", token),
		},
	})
	confMap.Clusters = append(confMap.Clusters, k8s_cli_api.NamedCluster{
		Name: name,
		Cluster: k8s_cli_api.Cluster{
			Server: server,
			CertificateAuthorityData: caData,
		},
	})
	return
}
