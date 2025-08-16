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
	"fmt"
	"net/http"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	crdv1 "k8s.io/dashboard/api/pkg/resource/customresourcedefinition/v1"
	"k8s.io/dashboard/client"
	"k8s.io/dashboard/types"
)

// CreateBackup creates a new Velero backup
func CreateBackup(request *http.Request, spec *BackupSpec) (*Backup, error) {
	// This GVR is not needed for REST client approach

	// Create unstructured object for the backup
	backup := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Backup",
			"metadata": map[string]interface{}{
				"name":      spec.Name,
				"namespace": spec.Namespace,
			},
			"spec": map[string]interface{}{
				"includedNamespaces": spec.IncludedNamespaces,
				"storageLocation":    spec.StorageLocation,
				"ttl":                spec.TTL,
			},
		},
	}

	// Add optional fields if provided
	if len(spec.ExcludedNamespaces) > 0 {
		backup.Object["spec"].(map[string]interface{})["excludedNamespaces"] = spec.ExcludedNamespaces
	}
	if len(spec.IncludedResources) > 0 {
		backup.Object["spec"].(map[string]interface{})["includedResources"] = spec.IncludedResources
	}
	if len(spec.ExcludedResources) > 0 {
		backup.Object["spec"].(map[string]interface{})["excludedResources"] = spec.ExcludedResources
	}
	if spec.LabelSelector != nil {
		backup.Object["spec"].(map[string]interface{})["labelSelector"] = spec.LabelSelector
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

	// Convert our backup object to JSON
	backupJSON, err := json.Marshal(backup.Object)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal backup: %s", err.Error())
	}

	// Create the backup via REST client
	result := restClient.Post().
		NamespaceIfScoped(spec.Namespace, customResourceDefinition.Spec.Scope == apiextensionsv1.NamespaceScoped).
		Resource(customResourceDefinition.Spec.Names.Plural).
		Body(backupJSON).
		Do(context.TODO())

	if result.Error() != nil {
		return nil, fmt.Errorf("Failed to create backup: %s", result.Error().Error())
	}

	// Get the raw response
	raw, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("Failed to get backup response: %s", err.Error())
	}

	// Parse the response to extract basic info
	var createdBackup map[string]interface{}
	if err := json.Unmarshal(raw, &createdBackup); err != nil {
		return nil, fmt.Errorf("Failed to parse backup response: %s", err.Error())
	}

	// Extract metadata
	metadata := createdBackup["metadata"].(map[string]interface{})

	// Convert to our Backup struct
	createdBackupResult := &Backup{
		ObjectMeta: types.ObjectMeta{
			Name:      metadata["name"].(string),
			Namespace: metadata["namespace"].(string),
		},
		TypeMeta: types.TypeMeta{
			Kind: "Backup",
		},
	}

	return createdBackupResult, nil
}

// BackupSpec represents the specification for creating a backup
type BackupSpec struct {
	Name               string                `json:"name"`
	Namespace          string                `json:"namespace"`
	IncludedNamespaces []string              `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces []string              `json:"excludedNamespaces,omitempty"`
	IncludedResources  []string              `json:"includedResources,omitempty"`
	ExcludedResources  []string              `json:"excludedResources,omitempty"`
	LabelSelector      *metav1.LabelSelector `json:"labelSelector,omitempty"`
	StorageLocation    string                `json:"storageLocation,omitempty"`
	TTL                string                `json:"ttl,omitempty"`
}
