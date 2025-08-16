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

package backup

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

// BackupDetail contains detailed information about a Velero backup.
type BackupDetail struct {
	ObjectMeta dashboardtypes.ObjectMeta `json:"objectMeta"`
	TypeMeta   dashboardtypes.TypeMeta   `json:"typeMeta"`

	// Backup specific fields
	Status       string `json:"status"`
	Phase        string `json:"phase"`
	StartTime    string `json:"startTime,omitempty"`
	CompletionTime string `json:"completionTime,omitempty"`
	Expiration   string `json:"expiration,omitempty"`
	StorageLocation string `json:"storageLocation,omitempty"`
	VolumeSnapshotLocation string `json:"volumeSnapshotLocation,omitempty"`
	
	// Progress and results
	TotalItems    int `json:"totalItems"`
	ItemsBackedUp int `json:"itemsBackedUp"`
	Progress      BackupProgress `json:"progress"`
	
	// Resource inclusion/exclusion
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	IncludedResources  []string `json:"includedResources,omitempty"`
	ExcludedResources  []string `json:"excludedResources,omitempty"`
	
	// Errors and warnings
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// BackupProgress represents the progress of a backup operation.
type BackupProgress struct {
	TotalItems    int `json:"totalItems"`
	ItemsBackedUp int `json:"itemsBackedUp"`
	ItemsFailed   int `json:"itemsFailed"`
}

// GetBackupDetail returns detailed information about a specific Velero backup.
func GetBackupDetail(request *http.Request, namespace *common.NamespaceQuery, name string) (*BackupDetail, error) {
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

	// Get the raw JSON data that contains the actual Velero backup information
	rawBackupData, err := getRawBackupData(apiExtClient, config, namespace, name)
	if err != nil {
		return nil, err
	}

	// Convert raw JSON to BackupDetail struct
	backupDetail, err := parseBackupDetail(rawBackupData)
	if err != nil {
		return nil, err
	}
	
	return backupDetail, nil
}

// getRawBackupData gets the raw JSON data for a specific Velero backup
func getRawBackupData(apiExtClient apiextensionsclientset.Interface, config *rest.Config, namespace *common.NamespaceQuery, name string) ([]byte, error) {
	// Get the backup CRD definition
	customResourceDefinition, err := apiExtClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "backups.velero.io", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Create REST client for the backup CRD
	restClient, err := crdv1.NewRESTClient(config, customResourceDefinition)
	if err != nil {
		return nil, err
	}

	// Get the raw backup data
	raw, err := restClient.Get().
		NamespaceIfScoped(namespace.ToRequestParam(), customResourceDefinition.Spec.Scope == apiextensionsv1.NamespaceScoped).
		Resource(customResourceDefinition.Spec.Names.Plural).
		Name(name).Do(context.TODO()).Raw()
	if err != nil {
		return nil, err
	}

	return raw, nil
}

// parseBackupDetail parses raw JSON data into a BackupDetail struct
func parseBackupDetail(rawData []byte) (*BackupDetail, error) {
	// Parse the raw JSON to extract Velero-specific fields
	var rawBackup map[string]interface{}
	if err := json.Unmarshal(rawData, &rawBackup); err != nil {
		return nil, err
	}

	// Extract metadata
	metadata := extractMetadata(rawBackup)
	
	// Create backup detail with basic info
	detail := &BackupDetail{
		ObjectMeta: metadata,
		TypeMeta: dashboardtypes.TypeMeta{
			Kind: "Backup",
		},
	}

	// Extract Velero-specific status information
	if status, ok := rawBackup["status"].(map[string]interface{}); ok {
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
		
		if expiration, ok := status["expiration"].(string); ok {
			detail.Expiration = expiration
		}
		
		// Extract progress information
		if progress, ok := status["progress"].(map[string]interface{}); ok {
			if totalItems, ok := progress["totalItems"].(float64); ok {
				detail.Progress.TotalItems = int(totalItems)
				detail.TotalItems = int(totalItems)
			}
			if itemsBackedUp, ok := progress["itemsBackedUp"].(float64); ok {
				detail.Progress.ItemsBackedUp = int(itemsBackedUp)
				detail.ItemsBackedUp = int(itemsBackedUp)
			}
		}
	}

	// Extract spec information
	if spec, ok := rawBackup["spec"].(map[string]interface{}); ok {
		if storageLocation, ok := spec["storageLocation"].(string); ok {
			detail.StorageLocation = storageLocation
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
func extractMetadata(rawBackup map[string]interface{}) dashboardtypes.ObjectMeta {
	metadata := dashboardtypes.ObjectMeta{}
	
	if meta, ok := rawBackup["metadata"].(map[string]interface{}); ok {
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
