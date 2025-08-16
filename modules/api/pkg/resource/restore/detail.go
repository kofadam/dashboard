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

package restore

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

// RestoreDetail contains detailed information about a Velero restore.
type RestoreDetail struct {
	ObjectMeta dashboardtypes.ObjectMeta `json:"objectMeta"`
	TypeMeta   dashboardtypes.TypeMeta   `json:"typeMeta"`

	// Restore specific fields
	Status         string `json:"status"`
	Phase          string `json:"phase"`
	StartTime      string `json:"startTime,omitempty"`
	CompletionTime string `json:"completionTime,omitempty"`
	BackupName     string `json:"backupName,omitempty"`
	
	// Progress and results
	TotalItems     int `json:"totalItems"`
	ItemsRestored  int `json:"itemsRestored"`
	Progress       RestoreProgress `json:"progress"`
	
	// Resource inclusion/exclusion
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	IncludedResources  []string `json:"includedResources,omitempty"`
	ExcludedResources  []string `json:"excludedResources,omitempty"`
	
	// Errors and warnings
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RestoreProgress represents the progress of a restore operation.
type RestoreProgress struct {
	TotalItems     int `json:"totalItems"`
	ItemsRestored  int `json:"itemsRestored"`
	ItemsFailed    int `json:"itemsFailed"`
}

// GetRestoreDetail returns detailed information about a specific Velero restore.
func GetRestoreDetail(request *http.Request, namespace *common.NamespaceQuery, name string) (*RestoreDetail, error) {
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

	// Get the raw JSON data that contains the actual Velero restore information
	rawRestoreData, err := getRawRestoreData(apiExtClient, config, namespace, name)
	if err != nil {
		return nil, err
	}

	// Convert raw JSON to RestoreDetail struct
	restoreDetail, err := parseRestoreDetail(rawRestoreData)
	if err != nil {
		return nil, err
	}
	
	return restoreDetail, nil
}

// getRawRestoreData gets the raw JSON data for a specific Velero restore
func getRawRestoreData(apiExtClient apiextensionsclientset.Interface, config *rest.Config, namespace *common.NamespaceQuery, name string) ([]byte, error) {
	// Get the restore CRD definition
	customResourceDefinition, err := apiExtClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "restores.velero.io", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Create REST client for the restore CRD
	restClient, err := crdv1.NewRESTClient(config, customResourceDefinition)
	if err != nil {
		return nil, err
	}

	// Get the raw restore data
	raw, err := restClient.Get().
		NamespaceIfScoped(namespace.ToRequestParam(), customResourceDefinition.Spec.Scope == apiextensionsv1.NamespaceScoped).
		Resource(customResourceDefinition.Spec.Names.Plural).
		Name(name).Do(context.TODO()).Raw()
	if err != nil {
		return nil, err
	}

	return raw, nil
}

// parseRestoreDetail parses raw JSON data into a RestoreDetail struct
func parseRestoreDetail(rawData []byte) (*RestoreDetail, error) {
	// Parse the raw JSON to extract Velero-specific fields
	var rawRestore map[string]interface{}
	if err := json.Unmarshal(rawData, &rawRestore); err != nil {
		return nil, err
	}

	// Extract metadata
	metadata := extractMetadata(rawRestore)
	
	// Create restore detail with basic info
	detail := &RestoreDetail{
		ObjectMeta: metadata,
		TypeMeta: dashboardtypes.TypeMeta{
			Kind: "Restore",
		},
	}

	// Extract Velero-specific status information
	if status, ok := rawRestore["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			detail.Phase = phase
			detail.Status = phase
		}
		
		if startTime, ok := status["startTimestamp"].(string); ok {
			detail.StartTime = startTime
		}
		
		if completionTime, ok := status["completionTimestamp"].(string); ok {
			detail.CompletionTime = completionTime
		}
		
		// Extract progress information
		if progress, ok := status["progress"].(map[string]interface{}); ok {
			if totalItems, ok := progress["totalItems"].(float64); ok {
				detail.Progress.TotalItems = int(totalItems)
				detail.TotalItems = int(totalItems)
			}
			if itemsRestored, ok := progress["itemsRestored"].(float64); ok {
				detail.Progress.ItemsRestored = int(itemsRestored)
				detail.ItemsRestored = int(itemsRestored)
			}
		}
	}

	// Extract spec information
	if spec, ok := rawRestore["spec"].(map[string]interface{}); ok {
		if backupName, ok := spec["backupName"].(string); ok {
			detail.BackupName = backupName
		}
		
		// Extract included/excluded namespaces
		if includedNS, ok := spec["includedNamespaces"].([]interface{}); ok {
			for _, ns := range includedNS {
				if nsStr, ok := ns.(string); ok {
					detail.IncludedNamespaces = append(detail.IncludedNamespaces, nsStr)
				}
			}
		}
	}

	return detail, nil
}

// extractMetadata extracts ObjectMeta from raw JSON
func extractMetadata(rawRestore map[string]interface{}) dashboardtypes.ObjectMeta {
	metadata := dashboardtypes.ObjectMeta{}
	
	if meta, ok := rawRestore["metadata"].(map[string]interface{}); ok {
		if name, ok := meta["name"].(string); ok {
			metadata.Name = name
		}
		if namespace, ok := meta["namespace"].(string); ok {
			metadata.Namespace = namespace
		}
		// Add more metadata fields as needed
	}
	
	return metadata
}
