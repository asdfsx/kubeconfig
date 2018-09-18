package main

import (
	"fmt"
	k8s_cli_api "k8s.io/client-go/tools/clientcmd/api/v1"
)

type Configmap struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Clusters   []struct {
		Name    string `json:"name"`
		Cluster struct {
			Server                   string `json:"Server"`
			CertificateAuthorityData string `json:"certificate_authority_data"`
		} `json:"clustrer"`
	} `json:"clusters,omitempty"`
	Contexts []struct {
		Name    string `json:"name"`
		Context struct {
			User    string `json:"user"`
			Cluster string `json:"cluster,omitempty"`
		} `json:"context"`
	} `json:"contexts,omitempty"`
	CurrentContext string `json:"current-context,omitempty"`
	Users          []struct {
		Name string `json:"name"`
		User struct {
			Token string `json:"token"`
		} `json:"user"`
	} `json:"users,omitempty"`
}

func generateConfigMap2(name string, token []byte) (confMap *k8s_cli_api.Config) {
	confMap = &k8s_cli_api.Config{}
	confMap.APIVersion = "v1"
	confMap.Kind = "Config"
	confMap.CurrentContext = name
	confMap.Contexts = append(confMap.Contexts, k8s_cli_api.NamedContext{
		Name: name,
		Context: k8s_cli_api.Context{
			AuthInfo: name,
		},
	})
	confMap.AuthInfos = append(confMap.AuthInfos, k8s_cli_api.NamedAuthInfo{
		Name: name,
		AuthInfo: k8s_cli_api.AuthInfo{
			Username: name,
			Token:    fmt.Sprintf("%s", token),
		},
	})
	return
}

func generateConfigMap(name string, token []byte) (confMap *Configmap) {
	confMap = &Configmap{
		ApiVersion:     "v1",
		Kind:           "Config",
		CurrentContext: name,
	}
	confMap.Contexts = append(confMap.Contexts,
		struct {
			Name    string `json:"name"`
			Context struct {
				User    string `json:"user"`
				Cluster string `json:"cluster,omitempty"`
			} `json:"context"`
		}{
			Name: name,
			Context: struct {
				User    string `json:"user"`
				Cluster string `json:"cluster,omitempty"`
			}{
				User: name,
			},
		})
	confMap.Users = append(confMap.Users,
		struct {
			Name string `json:"name"`
			User struct {
				Token string `json:"token"`
			} `json:"user"`
		}{
			Name: name,
			User: struct {
				Token string `json:"token"`
			}{
				Token: fmt.Sprintf("%s", token),
			},
		})
	return
}
