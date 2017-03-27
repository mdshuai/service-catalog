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

package tpr

import (
	"errors"
	"strconv"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/kubernetes"
)

type versioner struct {
	codec        runtime.Codec
	singularKind Kind
	listKind     Kind
	checkObject  func(runtime.Object) error
	cl           kubernetes.Interface
	defaultNS    string
}

// UpdateObject sets storage metadata into an API object. Returns an error if the object
// cannot be updated correctly. May return nil if the requested object does not need metadata
// from database.
func (t *versioner) UpdateObject(obj runtime.Object, resourceVersion uint64) error {
	if err := accessor.SetResourceVersion(obj, strconv.Itoa(int(resourceVersion))); err != nil {
		glog.Errorf("setting resource version (%s)", err)
		return err
	}
	name, err := accessor.Name(obj)
	if err != nil {
		glog.Errorf("getting name of the object (%s)", err)
		return err
	}
	// the Namespace function may return a nil error and an empty namespace. if it returns
	// a non-nil error and/or an empty namespace, use the default namespace
	ns, err := accessor.Namespace(obj)
	if err != nil || ns == "" {
		ns = t.defaultNS
	}

	data, err := runtime.Encode(t.codec, obj)
	if err != nil {
		glog.Errorf("encoding obj (%s)", err)
		return err
	}
	req := t.cl.Core().RESTClient().Put().AbsPath(
		"apis",
		groupName,
		tprVersion,
		"namespaces",
		ns,
		t.singularKind.URLName(),
		name,
	).Body(data)

	if err := req.Do().Error(); err != nil {
		glog.Errorf("error updating object %s (%s)", name, err)
		return err
	}
	return nil
}

func updateState(v storage.Versioner, st *objState, userUpdate storage.UpdateFunc) (runtime.Object, uint64, error) {
	ret, ttlPtr, err := userUpdate(st.obj, *st.meta)
	if err != nil {
		glog.Errorf("user update (%s)", err)
		return nil, 0, err
	}

	version, err := v.ObjectResourceVersion(ret)
	if err != nil {
		glog.Errorf("getting resource version (%s)", err)
		return nil, 0, err
	}
	if version != 0 {
		// We cannot store object with resourceVersion. We need to reset it.
		if err := v.UpdateObject(ret, 0); err != nil {
			glog.Errorf("updating object (%s)", err)
			return nil, 0, err
		}
	}
	var ttl uint64
	if ttlPtr != nil {
		ttl = *ttlPtr
	}
	return ret, ttl, nil
}

// UpdateList sets the resource version into an API list object. Returns an error if the object
// cannot be updated correctly. May return nil if the requested object does not need metadata
// from database.
func (t *versioner) UpdateList(obj runtime.Object, resourceVersion uint64) error {
	// ns, err := GetNamespace(obj)
	// if err != nil {
	// 	return err
	// }
	// if rvErr := GetAccessor().SetResourceVersion(
	// 	obj,
	// 	strconv.Itoa(int(resourceVersion)),
	// ); rvErr != nil {
	// 	return rvErr
	// }
	// unstruc, err := ToUnstructured(obj)
	// if err != nil {
	// 	return err
	// }
	// cl, err := GetResourceClient(t.cl, t.listKind, ns)
	// if err != nil {
	// 	return err
	// }
	// if _, err := cl.Update(unstruc); err != nil {
	// 	return err
	// }
	return errors.New("UpdateList not implemented")
}

// ObjectResourceVersion returns the resource version (for persistence) of the specified object.
// Should return an error if the specified object does not have a persistable version.
func (t *versioner) ObjectResourceVersion(obj runtime.Object) (uint64, error) {
	vsnStr, err := GetAccessor().ResourceVersion(obj)
	if err != nil {
		return 0, err
	}
	ret, err := strconv.ParseUint(vsnStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return ret, nil
}
