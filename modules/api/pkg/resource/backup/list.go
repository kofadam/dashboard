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
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	metricapi "k8s.io/dashboard/api/pkg/integration/metric/api"
	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/api/pkg/resource/dataselect"
	"k8s.io/dashboard/types"
)

// BackupList contains a list of Backups in the cluster.
type BackupList struct {
	ListMeta          types.ListMeta     `json:"listMeta"`
	CumulativeMetrics []metricapi.Metric `json:"cumulativeMetrics"`

	// Basic information about resources status on the list.
	Status common.ResourceStatus `json:"status"`

	// Unordered list of Backups.
	Backups []Backup `json:"backups"`

	// List of non-critical errors, that occurred during resource retrieval.
	Errors []error `json:"errors"`
}

type BackupStatusType string

const (
	// BackupStatusNew means the backup has been created but not processed.
	BackupStatusNew BackupStatusType = "New"
	// BackupStatusInProgress means the backup is currently running.
	BackupStatusInProgress BackupStatusType = "InProgress"
	// BackupStatusCompleted means the backup has completed successfully.
	BackupStatusCompleted BackupStatusType = "Completed"
	// BackupStatusFailed means the backup has failed.
	BackupStatusFailed BackupStatusType = "Failed"
)

type BackupStatus struct {
	// Short, machine understandable backup status code.
	Status BackupStatusType `json:"status"`
	// A human-readable description of the status of related backup.
	Message string `json:"message"`
	// Conditions describe the state of a backup after it finishes.
	Conditions []common.Condition `json:"conditions"`
}

// Backup is a presentation layer view of Velero Backup resource.
type Backup struct {
	ObjectMeta types.ObjectMeta `json:"objectMeta"`
	TypeMeta   types.TypeMeta   `json:"typeMeta"`

	// StorageLocation is the backup storage location name
	StorageLocation string `json:"storageLocation"`

	// TTL is the backup retention period
	TTL string `json:"ttl"`

	// IncludedNamespaces specifies which namespaces are included in the backup
	IncludedNamespaces []string `json:"includedNamespaces"`

	// ExcludedNamespaces specifies which namespaces are excluded from the backup
	ExcludedNamespaces []string `json:"excludedNamespaces"`

	// BackupStatus contains inferred backup status.
	BackupStatus BackupStatus `json:"backupStatus"`
}

// Helper function to convert unstructured backup to Backup struct
func toBackup(item *unstructured.Unstructured) Backup {
	// Create ObjectMeta manually from unstructured data
	objectMeta := types.ObjectMeta{}
	if name, found, _ := unstructured.NestedString(item.Object, "metadata", "name"); found {
		objectMeta.Name = name
	}
	if namespace, found, _ := unstructured.NestedString(item.Object, "metadata", "namespace"); found {
		objectMeta.Namespace = namespace
	}
	if uid, found, _ := unstructured.NestedString(item.Object, "metadata", "uid"); found {
		objectMeta.UID = ktypes.UID(uid)
	}
	if creationTime, found, _ := unstructured.NestedString(item.Object, "metadata", "creationTimestamp"); found {
		if parsedTime, err := time.Parse(time.RFC3339, creationTime); err == nil {
			objectMeta.CreationTimestamp = metav1.NewTime(parsedTime)
		}
	}

	backup := Backup{
		ObjectMeta: objectMeta,
		TypeMeta:   types.NewTypeMeta(types.ResourceKindBackup),
	}

	// Extract Velero-specific fields from unstructured data
	if storageLocation, found, _ := unstructured.NestedString(item.Object, "spec", "storageLocation"); found {
		backup.StorageLocation = storageLocation
	}
	
	if ttl, found, _ := unstructured.NestedString(item.Object, "spec", "ttl"); found {
		backup.TTL = ttl
	}

	if includedNS, found, _ := unstructured.NestedStringSlice(item.Object, "spec", "includedNamespaces"); found {
		backup.IncludedNamespaces = includedNS
	}

	if excludedNS, found, _ := unstructured.NestedStringSlice(item.Object, "spec", "excludedNamespaces"); found {
		backup.ExcludedNamespaces = excludedNS
	}

	// Extract status
	backup.BackupStatus = getBackupStatus(item)
	
	return backup
}

// Helper function to extract backup status from unstructured data
func getBackupStatus(item *unstructured.Unstructured) BackupStatus {
	status := BackupStatus{
		Status: BackupStatusNew,
		Message: "",
	}

	if phase, found, _ := unstructured.NestedString(item.Object, "status", "phase"); found {
		switch phase {
		case "New":
			status.Status = BackupStatusNew
		case "InProgress":
			status.Status = BackupStatusInProgress
		case "Completed":
			status.Status = BackupStatusCompleted
		case "Failed":
			status.Status = BackupStatusFailed
		default:
			status.Status = BackupStatusNew
		}
	}

	if message, found, _ := unstructured.NestedString(item.Object, "status", "failureReason"); found {
		status.Message = message
	}

	return status
}

// GetBackupList returns a list of all Backups in the cluster using dynamic client.
func GetBackupList(dynamicClient dynamic.Interface, namespace string, dsQuery *dataselect.DataSelectQuery) (*BackupList, error) {
	klog.V(4).Infof("Getting list of all backups in namespace %s", namespace)

	// Get backups using dynamic client
	var backupList *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		// Get all backups across all namespaces
		backupList, err = dynamicClient.Resource(BackupGVR).List(context.TODO(), metav1.ListOptions{})
	} else {
		// Get backups in specific namespace
		backupList, err = dynamicClient.Resource(BackupGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	}

	if err != nil {
		return nil, err
	}

	return toBackupList(backupList.Items, dsQuery), nil
}

// toBackupList converts unstructured backup list to BackupList
func toBackupList(backups []unstructured.Unstructured, dsQuery *dataselect.DataSelectQuery) *BackupList {
	backupList := &BackupList{
		Backups:  make([]Backup, 0),
		ListMeta: types.ListMeta{TotalItems: len(backups)},
		Errors:   []error{},
	}

	// Apply data selection (filtering, sorting, pagination)
	backupCells, filteredTotal := dataselect.GenericDataSelectWithFilter(ToCells(backups), dsQuery)
	backups = FromCells(backupCells)
	backupList.ListMeta = types.ListMeta{TotalItems: filteredTotal}

	// Convert to Backup format
	for _, backup := range backups {
		backupList.Backups = append(backupList.Backups, toBackup(&backup))
	}

	// Set status summary
	backupList.Status = getBackupListStatus(backups)

	return backupList
}

// getBackupConditions extracts conditions from backup status
func getBackupConditions(backup *unstructured.Unstructured) []common.Condition {
	conditions := []common.Condition{}
	if conditionsInterface, found, _ := unstructured.NestedSlice(backup.Object, "status", "conditions"); found {
		for _, conditionInterface := range conditionsInterface {
			if condition, ok := conditionInterface.(map[string]interface{}); ok {
				if condType, found := condition["type"].(string); found {
					if condStatus, found := condition["status"].(string); found {
						newCondition := common.Condition{
							Type:   condType,
							Status: v1.ConditionStatus(condStatus),
						}
						if message, found := condition["message"].(string); found {
							newCondition.Message = message
						}
						conditions = append(conditions, newCondition)
					}
				}
			}
		}
	}
	return conditions
}