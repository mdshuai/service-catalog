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
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	kubeclientfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	core "k8s.io/client-go/testing"
)

var (
	fakeClientset   *kubeclientfake.Clientset
	getCallCount    int
	createCallCount int
	createCallArgs  []string
	getCallArgs     []string
)

func setup(getFn, createFn func(core.Action) (bool, runtime.Object, error)) {
	getCallCount = 0
	createCallCount = 0
	getCallArgs = []string{}
	createCallArgs = []string{}

	fakeClientset = &kubeclientfake.Clientset{}

	fakeClientset.AddReactor("get", "thirdpartyresources", getFn)

	fakeClientset.AddReactor("create", "thirdpartyresources", createFn)
}

//make sure all resources types are installed
func TestInstallTypesAllResources(t *testing.T) {
	setup(
		func(core.Action) (bool, runtime.Object, error) {
			getCallCount++
			if getCallCount > len(thirdPartyResources) {
				return true, &v1beta1.ThirdPartyResource{}, nil
			}

			return true, nil, errors.New("Resource not found")
		},
		func(core.Action) (bool, runtime.Object, error) {
			createCallCount++
			return true, nil, nil
		},
	)

	installer := NewInstaller(fakeClientset.Extensions().ThirdPartyResources())
	installer.InstallTypes()

	if createCallCount != len(thirdPartyResources) {
		t.Errorf("Error: The number of Third Party Resources installed is not 4")
	}
}

//make sure to skip resource that is already installed
func TestInstallTypesResourceExisted(t *testing.T) {
	setup(
		func(core.Action) (bool, runtime.Object, error) {
			getCallCount++
			if getCallCount == 1 {
				return true, &serviceBrokerTPR, nil
			} else if getCallCount > len(thirdPartyResources) {
				return true, &v1beta1.ThirdPartyResource{}, nil
			}

			return true, nil, errors.New("Resource not found")
		},
		func(action core.Action) (bool, runtime.Object, error) {
			createCallCount++
			createCallArgs = append(createCallArgs, action.(core.CreateAction).GetObject().(*v1beta1.ThirdPartyResource).Name)
			return true, nil, nil
		},
	)

	installer := NewInstaller(fakeClientset.Extensions().ThirdPartyResources())
	installer.InstallTypes()

	if createCallCount != len(thirdPartyResources)-1 {
		t.Errorf("Failed to skip 1 installed Third Party Resource")
	}

	for _, name := range createCallArgs {
		if name == serviceBrokerTPR.Name {
			t.Errorf("Failed to skip installing 'broker' as Third Party Resource as it already existed")
		}
	}
}

//make sure all errors are received for all failed install
func TestInstallTypesErrors(t *testing.T) {
	setup(
		func(core.Action) (bool, runtime.Object, error) {
			getCallCount++
			if getCallCount > len(thirdPartyResources) {
				return true, &v1beta1.ThirdPartyResource{}, nil
			}

			return true, nil, errors.New("Resource not found")
		},
		func(core.Action) (bool, runtime.Object, error) {
			createCallCount++
			if createCallCount <= 2 {
				return true, nil, errors.New("Error " + strconv.Itoa(createCallCount))
			}
			return true, nil, nil
		},
	)

	installer := NewInstaller(fakeClientset.Extensions().ThirdPartyResources())
	err := installer.InstallTypes()

	errStr := err.Error()
	if !strings.Contains(errStr, "Error 1") && !strings.Contains(errStr, "Error 2") {
		t.Errorf("Failed to receive correct errors during installation of Third Party Resource concurrently, error received: %s", errStr)
	}
}

//make sure we don't poll on resource that was failed on install
func TestInstallTypesPolling(t *testing.T) {
	setup(
		func(action core.Action) (bool, runtime.Object, error) {
			getCallCount++
			if getCallCount > len(thirdPartyResources) {
				getCallArgs = append(getCallArgs, action.(core.GetAction).GetName())
				return true, &v1beta1.ThirdPartyResource{}, nil
			}

			return true, nil, errors.New("Resource not found")
		},
		func(action core.Action) (bool, runtime.Object, error) {
			createCallCount++
			name := action.(core.CreateAction).GetObject().(*v1beta1.ThirdPartyResource).Name
			if name == serviceBrokerTPR.Name || name == serviceInstanceTPR.Name {
				return true, nil, errors.New("Error creating TPR")
			}
			return true, nil, nil
		},
	)

	installer := NewInstaller(fakeClientset.Extensions().ThirdPartyResources())
	installer.InstallTypes()

	for _, name := range getCallArgs {
		if name == serviceBrokerTPR.Name || name == serviceInstanceTPR.Name {
			t.Errorf("Failed to skip polling for resource that failed to install")
		}
	}
}
