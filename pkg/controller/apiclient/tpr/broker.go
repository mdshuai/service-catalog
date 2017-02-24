/*
Copyright 2016 The Kubernetes Authors.

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

package tpr

import (
	"errors"
	"log"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog"
	"github.com/kubernetes-incubator/service-catalog/pkg/controller/util"
	"github.com/kubernetes-incubator/service-catalog/pkg/controller/watch"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type brokerClient struct {
	watcher *watch.Watcher
}

func newBrokerClient(watcher *watch.Watcher) *brokerClient {
	return &brokerClient{watcher: watcher}
}

func (c *brokerClient) List() ([]*servicecatalog.Broker, error) {
	l, err := c.watcher.GetResourceClient(watch.ServiceBroker, "default").List(&v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var ret []*servicecatalog.Broker
	for _, i := range l.(*runtime.UnstructuredList).Items {
		var tmp servicecatalog.Broker
		err := util.TPRObjectToSCObject(i, &tmp)
		if err != nil {
			log.Printf("Failed to convert object: %v\n", err)
			return nil, err
		}
		ret = append(ret, &tmp)
	}
	return ret, nil
}

func (c *brokerClient) Get(name string) (*servicecatalog.Broker, error) {
	log.Printf("Getting broker: %s\n", name)

	sb, err := c.watcher.GetResourceClient(watch.ServiceBroker, "default").Get(name)
	if err != nil {
		return nil, err
	}
	var tmp servicecatalog.Broker
	err = util.TPRObjectToSCObject(sb, &tmp)
	if err != nil {
		return nil, err
	}
	return &tmp, nil
}

func (c *brokerClient) Create(broker *servicecatalog.Broker) (*servicecatalog.Broker, error) {
	broker.Kind = watch.ServiceBrokerKind
	broker.APIVersion = watch.FullAPIVersion
	tprObj, err := util.SCObjectToTPRObject(broker)
	if err != nil {
		log.Printf("Failed to convert object %#v : %v", broker, err)
		return nil, err
	}
	tprObj.SetName(broker.Name)
	// TODO: Are brokers always in default namespace, if not, need to tweak this.
	log.Printf("Creating Broker: %s\n", broker.Name)
	c.watcher.GetResourceClient(watch.ServiceBroker, "default").Create(tprObj)

	// krancour: Ideally the broker we return is a translation of the updated 3pr
	// as read back from k8s. It doesn't seem worth going through the trouble
	// right now since 3pr storage will be removed eventually. This will at least
	// work well enough in the meantime.
	return broker, nil
}

func (*brokerClient) Update(broker *servicecatalog.Broker) (*servicecatalog.Broker, error) {
	return nil, errors.New("Not implemented yet")
}

func (*brokerClient) Delete(id string) error {
	return errors.New("Not implemented yet")
}
