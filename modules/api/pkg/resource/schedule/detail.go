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
	"net/http"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"k8s.io/dashboard/api/pkg/resource/common"
	crdv1 "k8s.io/dashboard/api/pkg/resource/customresourcedefinition/v1"
	"k8s.io/dashboard/client"
	dashboardtypes "k8s.io/dashboard/types"
)

// ScheduleDetail contains detailed information about a Velero schedule.
type ScheduleDetail struct {
	ObjectMeta      dashboardtypes.ObjectMeta `json:"objectMeta"`
	TypeMeta        dashboardtypes.TypeMeta   `json:"typeMeta"`
	Schedule        string                    `json:"schedule"`
	LastBackupTime  string                    `json:"lastBackupTime,omitempty"`
	Phase           string                    `json:"phase,omitempty"`
	Status          string                    `json:"status,omitempty"`
	ValidationError string                    `json:"validationError,omitempty"`
}

// GetScheduleDetail returns detailed information about a specific Velero schedule
func GetScheduleDetail(request *http.Request, namespace *common.NamespaceQuery, name string) (*ScheduleDetail, error) {
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

	// Get raw schedule data
	rawScheduleData, err := getRawScheduleData(apiExtClient, config, namespace, name)
	if err != nil {
		return nil, err
	}

	// Parse the raw data into schedule detail
	scheduleDetail, err := parseScheduleDetail(rawScheduleData)
	if err != nil {
		return nil, err
	}

	return scheduleDetail, nil
}

// getRawScheduleData gets the raw JSON data for a specific Velero schedule
func getRawScheduleData(apiExtClient apiextensionsclientset.Interface, config *rest.Config, namespace *common.NamespaceQuery, name string) ([]byte, error) {
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

	// Get the raw schedule data
	raw, err := restClient.Get().
		NamespaceIfScoped(namespace.ToRequestParam(), customResourceDefinition.Spec.Scope == apiextensionsv1.NamespaceScoped).
		Resource(customResourceDefinition.Spec.Names.Plural).
		Name(name).Do(context.TODO()).Raw()
	if err != nil {
		return nil, err
	}

	return raw, nil
}

// parseScheduleDetail parses raw JSON data into a ScheduleDetail struct
func parseScheduleDetail(rawData []byte) (*ScheduleDetail, error) {
	// Parse the raw JSON to extract Velero-specific fields
	var rawSchedule map[string]interface{}
	if err := json.Unmarshal(rawData, &rawSchedule); err != nil {
		return nil, err
	}

	// Extract metadata
	metadata := extractMetadata(rawSchedule)

	// Create schedule detail with basic info
	detail := &ScheduleDetail{
		ObjectMeta: metadata,
		TypeMeta: dashboardtypes.TypeMeta{
			Kind: "Schedule",
		},
	}

	// Extract Velero-specific spec information
	if spec, ok := rawSchedule["spec"].(map[string]interface{}); ok {
		if schedule, ok := spec["schedule"].(string); ok {
			detail.Schedule = schedule
		}
	}

	// Extract Velero-specific status information
	if status, ok := rawSchedule["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			detail.Phase = phase
			detail.Status = phase
		}
		if lastBackupTime, ok := status["lastBackupTime"].(string); ok {
			detail.LastBackupTime = lastBackupTime
		}
		if validationErrors, ok := status["validationErrors"].([]interface{}); ok && len(validationErrors) > 0 {
			if validationError, ok := validationErrors[0].(string); ok {
				detail.ValidationError = validationError
			}
		}
	}

	return detail, nil
}

// extractMetadata extracts standard Kubernetes metadata from raw JSON
func extractMetadata(rawSchedule map[string]interface{}) dashboardtypes.ObjectMeta {
	metadata := dashboardtypes.ObjectMeta{}

	if meta, ok := rawSchedule["metadata"].(map[string]interface{}); ok {
		if name, ok := meta["name"].(string); ok {
			metadata.Name = name
		}
		if namespace, ok := meta["namespace"].(string); ok {
			metadata.Namespace = namespace
		}
	}

	return metadata
}
