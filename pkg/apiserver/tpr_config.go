/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/service-catalog/pkg/registry/servicecatalog/server"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/storage"
	clientset "k8s.io/client-go/kubernetes"
)

// tprConfig is the configuration needed to run the API server in TPR storage mode
type tprConfig struct {
	cl              clientset.Interface
	genericConfig   *genericapiserver.Config
	globalNamespace string
	storageFactory  storage.StorageFactory
}

// NewTPRConfig returns a new Config for a server that is backed by TPR storage
func NewTPRConfig(
	cl clientset.Interface,
	genericCfg *genericapiserver.Config,
	globalNS string,
	factory storage.StorageFactory,
) Config {
	return &tprConfig{
		cl:              cl,
		genericConfig:   genericCfg,
		globalNamespace: globalNS,
		storageFactory:  factory,
	}
}

// Complete fills in the remaining fields of t and returns a completed config
func (t *tprConfig) Complete() CompletedConfig {
	completeGenericConfig(t.genericConfig)
	return &completedTPRConfig{
		cl:        t.cl,
		tprConfig: t,
		// Not every API group compiled in is necessarily enabled by the operator
		// at runtime.
		//
		// Install the API resource config source, which describes versions of
		// which API groups are enabled.
		apiResourceConfigSource: DefaultAPIResourceConfigSource(),
		factory:                 t.storageFactory,
	}
}

// CompletedTPRConfig is the completed version of the TPR config. It can be used to create a
// new server, ready to be run
type completedTPRConfig struct {
	cl clientset.Interface
	*tprConfig
	apiResourceConfigSource storage.APIResourceConfigSource
	factory                 storage.StorageFactory
}

// NewServer returns a new service catalog server, that is ready for execution
func (c *completedTPRConfig) NewServer() (*ServiceCatalogAPIServer, error) {
	s, err := createSkeletonServer(c.tprConfig.genericConfig)
	if err != nil {
		return nil, err
	}
	glog.V(4).Infoln("Created skeleton API server. Installing API groups")

	// JPEELER
	// roFactory := tprRESTOptionsFactory{
	// 	storageFactory: c.factory,
	// }
	providers := restStorageProviders(c.globalNamespace, server.StorageTypeTPR, c.cl)
	for _, provider := range providers {
		groupInfo, err := provider.NewRESTStorage(
			c.apiResourceConfigSource, // genericapiserver.APIResourceConfigSource
			c.genericConfig.RESTOptionsGetter,
		)
		if IsErrAPIGroupDisabled(err) {
			glog.Warningf("Skipping API group %v because it is not enabled", provider.GroupName())
			continue
		} else if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Installing API group %v", provider.GroupName())
		if err := s.GenericAPIServer.InstallAPIGroup(groupInfo); err != nil {
			glog.Fatalf("Error installing API group %v: %v", provider.GroupName(), err)
		}
	}
	glog.Infoln("Finished installing API groups")
	return s, nil
}
