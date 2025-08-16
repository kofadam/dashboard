// Copyright 2017 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schedule

import (
	"context"
	"fmt"
	"net/http"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crdv1 "k8s.io/dashboard/api/pkg/resource/customresourcedefinition/v1"
	"k8s.io/dashboard/client"
)

// DeleteSchedule deletes a Velero schedule
func DeleteSchedule(request *http.Request, namespace, name string) error {
	// Get API extensions client for CRD operations
	apiExtClient, err := client.APIExtensionsClient(request)
	if err != nil {
		return err
	}

	// Get REST config for custom resource operations
	config, err := client.Config(request)
	if err != nil {
		return err
	}

	// Get the schedule CRD definition
	customResourceDefinition, err := apiExtClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "schedules.velero.io", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Create REST client for the schedule CRD
	restClient, err := crdv1.NewRESTClient(config, customResourceDefinition)
	if err != nil {
		return err
	}

	// Delete the schedule via REST client
	result := restClient.Delete().
		NamespaceIfScoped(namespace, customResourceDefinition.Spec.Scope == apiextensionsv1.NamespaceScoped).
		Resource(customResourceDefinition.Spec.Names.Plural).
		Name(name).
		Do(context.TODO())

	if result.Error() != nil {
		return fmt.Errorf("Failed to delete schedule: %s", result.Error().Error())
	}

	return nil
}
