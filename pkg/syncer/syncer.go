// Package syncer provides the logic used by command line tool cloudstack-csi-sc-syncer.
//
// It provides functions to synchronize CloudStack disk offerings
// to Kubernetes storage classes.
package syncer

import (
	"context"
	"fmt"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
)

// Config holds the syncer tool configuration.
type Config struct {
	Agent            string
	CloudStackConfig string
	KubeConfig       string
	Label            string
	NamePrefix       string
	Delete           bool
	VolumeExpansion  bool
	NodeName         string
	AddAllowedTopology bool
}

// Syncer has a function Run which synchronizes CloudStack
// disk offerings to Kubernetes Storage classes.
type Syncer interface {
	Run(context.Context) error
}

// syncer is Syncer implementation.
type syncer struct {
	k8sClient       *kubernetes.Clientset
	csClient        *cloudstack.CloudStackClient
	labelsSet       labels.Set
	namePrefix      string
	delete          bool
	volumeExpansion bool
	csConnector     cloud.Cloud
	nodeName        string
	addAllowedTopology bool
}

func createK8sClient(kubeconfig, agent string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if kubeconfig == "-" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	config.UserAgent = agent

	return kubernetes.NewForConfig(config)
}

func createCloudStackClient(cloudstackconfig string) (*cloudstack.CloudStackClient, error) {
	config, err := cloud.ReadConfig(cloudstackconfig)
	if err != nil {
		return nil, err
	}
	client := cloudstack.NewAsyncClient(config.APIURL, config.APIKey, config.SecretKey, config.VerifySSL)

	return client, nil
}

func createCSConnector(cloudstackconfig string) (cloud.Cloud, error) {
	config, err := cloud.ReadConfig(cloudstackconfig)
	if err != nil {
		return nil, err
	}

	csConnector := cloud.New(config)

	return csConnector, nil
}

func createLabelsSet(label string) labels.Set {
	m := make(map[string]string)
	if len(label) > 0 {
		parts := strings.SplitN(label, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		m[key] = value
	}

	return m
}

// New creates a new Syncer instance.
func New(config Config) (Syncer, error) {
	k8sClient, err := createK8sClient(config.KubeConfig, config.Agent)
	if err != nil {
		return nil, fmt.Errorf("cannot create Kubernetes client: %w", err)
	}
	csClient, err := createCloudStackClient(config.CloudStackConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create CloudStack client: %w", err)
	}

	csConnector, err := createCSConnector(config.CloudStackConfig)
	if err != nil {
			return nil, fmt.Errorf("cannot create CS connector interface: %w", err)
		}


	return syncer{
		k8sClient:       k8sClient,
		csClient:        csClient,
		labelsSet:       createLabelsSet(config.Label),
		namePrefix:      config.NamePrefix,
		delete:          config.Delete,
		volumeExpansion: config.VolumeExpansion,
		csConnector:        csConnector,
		nodeName:           config.NodeName,
		addAllowedTopology: config.AddAllowedTopology,
	}, nil
}
