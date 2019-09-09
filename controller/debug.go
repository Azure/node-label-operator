package controller

import (
	"io/ioutil"
	"os"

	config "github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
)

func NewCloudProvider(configFile string) (c *config.AzureConfig, e error) {
	azureConfig := config.AzureConfig{}

	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		glog.Errorf("Read file (%s) error: %+v", configFile, err)
		return nil, err
	}
	if err = yaml.Unmarshal(bytes, &azureConfig); err != nil {
		glog.Errorf("Unmarshall error: %v", err)
		return nil, err
	}

	return &azureConfig, nil
}

func Debug() (*config.AzureConfig, error) {
	path := os.Getenv("HOME") + "/.azure/personal/azure.json"
	config, err := NewCloudProvider(path)
	if err != nil {
		return nil, err
	}

	return config, nil
}
