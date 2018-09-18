package main

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"testing"
)

func TestConfigmap(t *testing.T) {
	configmap := generateConfigMap("test", []byte("test"))

	fmt.Println(configmap)
	result, err := json.Marshal(configmap)
	if err != nil {
		t.Fatal(err)
	}
	result, err = yaml.JSONToYAML(result)
	fmt.Printf("%s\n", result)
}
