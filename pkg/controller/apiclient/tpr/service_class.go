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
	"log"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog"
	"github.com/kubernetes-incubator/service-catalog/pkg/controller/util"
	"github.com/kubernetes-incubator/service-catalog/pkg/controller/watch"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type serviceClassClient struct {
	watcher *watch.Watcher
}

func newServiceClassClient(watcher *watch.Watcher) *serviceClassClient {
	return &serviceClassClient{watcher: watcher}
}

func (c *serviceClassClient) Get(name string) (*servicecatalog.ServiceClass, error) {
	si, err := c.watcher.GetResourceClient(watch.ServiceClass, "default").Get(name)
	if err != nil {
		return nil, err
	}
	var tmp servicecatalog.ServiceClass
	err = util.TPRObjectToSCObject(si, &tmp)
	if err != nil {
		return nil, err
	}
	return &tmp, nil
}

func (c *serviceClassClient) List() ([]*servicecatalog.ServiceClass, error) {
	l, err := c.watcher.GetResourceClient(watch.ServiceClass, "default").List(&v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list service types: %v\n", err)
		return nil, err
	}
	var lst []*servicecatalog.ServiceClass
	for _, i := range l.(*runtime.UnstructuredList).Items {
		var tmp servicecatalog.ServiceClass
		err := util.TPRObjectToSCObject(i, &tmp)
		if err != nil {
			log.Printf("Failed to convert object: %v\n", err)
			return nil, err
		}
		lst = append(lst, &tmp)
	}
	return lst, nil

}

func (c *serviceClassClient) Create(sc *servicecatalog.ServiceClass) (*servicecatalog.ServiceClass, error) {
	sc.Kind = watch.ServiceClassKind
	sc.APIVersion = watch.FullAPIVersion
	tprObj, err := util.SCObjectToTPRObject(sc)
	if err != nil {
		log.Printf("Failed to convert object %#v : %v", sc, err)
		return nil, err
	}
	tprObj.SetName(sc.Name)
	log.Printf("Creating k8sobject:\n%v\n", tprObj)
	_, err = c.watcher.GetResourceClient(watch.ServiceClass, "default").Create(tprObj)
	if err != nil {
		return nil, err
	}
	// krancour: Ideally the instance we return is a translation of the updated
	// 3pr as read back from k8s. It doesn't seem worth going through the trouble
	// right now since 3pr storage will be removed eventually. This will at least
	// work well enough in the meantime.
	return sc, nil
}
