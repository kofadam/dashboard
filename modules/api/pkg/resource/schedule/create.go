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
	"encoding/json"
	"fmt"
	"net/http"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	crdv1 "k8s.io/dashboard/api/pkg/resource/customresourcedefinition/v1"
	"k8s.io/dashboard/client"
	"k8s.io/dashboard/types"
)

// CreateSchedule creates a new Velero schedule
func CreateSchedule(request *http.Request, spec *ScheduleSpec) (*Schedule, error) {
	// Create unstructured object for the schedule
	schedule := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Schedule",
			"metadata": map[string]interface{}{
				"name":      spec.Name,
				"namespace": spec.Namespace,
			},
			"spec": map[string]interface{}{
				"schedule": spec.Schedule,
				"template": map[string]interface{}{
					"includedNamespaces": spec.IncludedNamespaces,
					"storageLocation":    spec.StorageLocation,
					"ttl":                spec.TTL,
				},
			},
		},
	}

	// Add optional fields if provided
	if len(spec.ExcludedNamespaces) > 0 {
		schedule.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["excludedNamespaces"] = spec.ExcludedNamespaces
	}
	if len(spec.IncludedResources) > 0 {
		schedule.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["includedResources"] = spec.IncludedResources
	}
	if len(spec.ExcludedResources) > 0 {
		schedule.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["excludedResources"] = spec.ExcludedResources
	}
	if spec.LabelSelector != nil {
		schedule.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["labelSelector"] = spec.LabelSelector
	}

	// Get API extensions client for CRD operations
	apiExtClient, err := client.APIExtensionsClient(request)
	if err != nil {
		return nil, err
	}

	// Get REST config for custom resource operations
	config, err := client.Config(request)
	if err != nil {
		return nil, err
	}

	// Get the schedule CRD definition
	customResourceDefinition, err := apiExtClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "schedules.velero.io", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Create REST client for the schedule CRD
	restClient, err := crdv1.NewRESTClient(config, customResourceDefinition)
	if err != nil {
		return nil, err
	}

	// Convert our schedule object to JSON
	scheduleJSON, err := json.Marshal(schedule.Object)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal schedule: %s", err.Error())
	}

	// Create the schedule via REST client
	result := restClient.Post().
		NamespaceIfScoped(spec.Namespace, customResourceDefinition.Spec.Scope == apiextensionsv1.NamespaceScoped).
		Resource(customResourceDefinition.Spec.Names.Plural).
		Body(scheduleJSON).
		Do(context.TODO())

	if result.Error() != nil {
		return nil, fmt.Errorf("Failed to create schedule: %s", result.Error().Error())
	}

	// Get the raw response
	raw, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("Failed to get schedule response: %s", err.Error())
	}

	// Parse the response to extract basic info
	var createdSchedule map[string]interface{}
	if err := json.Unmarshal(raw, &createdSchedule); err != nil {
		return nil, fmt.Errorf("Failed to parse schedule response: %s", err.Error())
	}

	// Extract metadata
	metadata := createdSchedule["metadata"].(map[string]interface{})

	// Convert to our Schedule struct
	createdScheduleResult := &Schedule{
		ObjectMeta: types.ObjectMeta{
			Name:      metadata["name"].(string),
			Namespace: metadata["namespace"].(string),
		},
		TypeMeta: types.TypeMeta{
			Kind: "Schedule",
		},
	}

	return createdScheduleResult, nil
}

// ScheduleSpec represents the specification for creating a schedule
type ScheduleSpec struct {
	Name               string                `json:"name"`
	Namespace          string                `json:"namespace"`
	Schedule           string                `json:"schedule"` // Cron schedule expression
	IncludedNamespaces []string              `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces []string              `json:"excludedNamespaces,omitempty"`
	IncludedResources  []string              `json:"includedResources,omitempty"`
	ExcludedResources  []string              `json:"excludedResources,omitempty"`
	LabelSelector      *metav1.LabelSelector `json:"labelSelector,omitempty"`
	StorageLocation    string                `json:"storageLocation,omitempty"`
	TTL                string                `json:"ttl,omitempty"`
}
