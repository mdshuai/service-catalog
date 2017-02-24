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

type instanceClient struct {
	watcher *watch.Watcher
	ns      string
}

func newInstanceClient(watcher *watch.Watcher, ns string) *instanceClient {
	return &instanceClient{watcher: watcher, ns: ns}
}

func (c *instanceClient) List() ([]*servicecatalog.Instance, error) {
	l, err := c.watcher.GetResourceClient(watch.ServiceInstance, c.ns).List(&v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var ret []*servicecatalog.Instance
	for _, i := range l.(*runtime.UnstructuredList).Items {
		var tmp servicecatalog.Instance
		err := util.TPRObjectToSCObject(i, &tmp)
		if err != nil {
			log.Printf("Failed to convert object: %v\n", err)
			return nil, err
		}
		ret = append(ret, &tmp)
	}
	return ret, nil
}

func (c *instanceClient) Get(name string) (*servicecatalog.Instance, error) {
	si, err := c.watcher.GetResourceClient(watch.ServiceInstance, c.ns).Get(name)
	if err != nil {
		return nil, err
	}
	var tmp servicecatalog.Instance
	err = util.TPRObjectToSCObject(si, &tmp)
	if err != nil {
		return nil, err
	}
	return &tmp, nil
}

func (c *instanceClient) Create(si *servicecatalog.Instance) (*servicecatalog.Instance, error) {
	si.Kind = watch.ServiceInstanceKind
	si.APIVersion = watch.FullAPIVersion
	tprObj, err := util.SCObjectToTPRObject(si)
	if err != nil {
		log.Printf("Failed to convert object %#v : %v", si, err)
		return nil, err
	}
	tprObj.SetName(si.Name)
	log.Printf("Creating k8sobject:\n%v\n", tprObj)
	_, err = c.watcher.GetResourceClient(watch.ServiceInstance, c.ns).Create(tprObj)
	if err != nil {
		return nil, err
	}
	// krancour: Ideally the instance we return is a translation of the updated
	// 3pr as read back from k8s. It doesn't seem worth going through the trouble
	// right now since 3pr storage will be removed eventually. This will at least
	// work well enough in the meantime.
	return si, nil
}

func (c *instanceClient) Update(si *servicecatalog.Instance) (*servicecatalog.Instance, error) {
	si.Kind = watch.ServiceInstanceKind
	si.APIVersion = watch.FullAPIVersion
	tprObj, err := util.SCObjectToTPRObject(si)
	if err != nil {
		log.Printf("Failed to convert object %#v : %v", si, err)
		return nil, err
	}
	tprObj.SetName(si.Name)
	_, err = c.watcher.GetResourceClient(watch.ServiceInstance, "default").Update(tprObj)
	if err != nil {
		return nil, err
	}
	// krancour: Ideally the instance we return is a translation of the updated
	// 3pr as read back from k8s. It doesn't seem worth going through the trouble
	// right now since 3pr storage will be removed soon. This will at least work
	// well enough in the meantime.
	return si, nil
}

func (*instanceClient) Delete(string) error {
	return errors.New("Not implemented yet")
}
