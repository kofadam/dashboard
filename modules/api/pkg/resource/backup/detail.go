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
	"k8s.io/client-go/dynamic"
)

// BackupDetail is a presentation layer view of Velero Backup resource.
type BackupDetail struct {
	// Extends list item structure.
	Backup `json:",inline"`

	// StartTimestamp indicates when the backup started
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`

	// CompletionTimestamp indicates when the backup completed
	CompletionTimestamp *metav1.Time `json:"completionTimestamp,omitempty"`

	// Progress contains backup progress information
	Progress *BackupProgress `json:"progress,omitempty"`

	// List of non-critical errors, that occurred during resource retrieval.
	Errors []error `json:"errors"`
}

// BackupProgress contains backup progress information
type BackupProgress struct {
	TotalItems      *int `json:"totalItems,omitempty"`
	ItemsBackedUp   *int `json:"itemsBackedUp,omitempty"`
	BytesBackedUp   *int64 `json:"bytesBackedUp,omitempty"`
}

// GetBackupDetail gets backup details using dynamic client.
func GetBackupDetail(dynamicClient dynamic.Interface, namespace, name string) (*BackupDetail, error) {
	// Get backup using dynamic client
	backup, err := dynamicClient.Resource(BackupGVR).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	backupDetail := &BackupDetail{
		Backup: toBackup(backup),
	}

	// Extract additional detail fields
	if startTime, found, _ := unstructured.NestedString(backup.Object, "status", "startTimestamp"); found {
		if parsedTime, err := time.Parse(time.RFC3339, startTime); err == nil {
			metaTime := metav1.NewTime(parsedTime)
			backupDetail.StartTimestamp = &metaTime
		}
	}

	if completionTime, found, _ := unstructured.NestedString(backup.Object, "status", "completionTimestamp"); found {
		if parsedTime, err := time.Parse(time.RFC3339, completionTime); err == nil {
			metaTime := metav1.NewTime(parsedTime)
			backupDetail.CompletionTimestamp = &metaTime
		}
	}

	// Extract progress information
	backupDetail.Progress = extractBackupProgress(backup)

	return backupDetail, nil
}

// Helper function to extract backup progress from unstructured data
func extractBackupProgress(backup *unstructured.Unstructured) *BackupProgress {
	progress := &BackupProgress{}

	if totalItems, found, _ := unstructured.NestedInt64(backup.Object, "status", "progress", "totalItems"); found {
		items := int(totalItems)
		progress.TotalItems = &items
	}

	if itemsBackedUp, found, _ := unstructured.NestedInt64(backup.Object, "status", "progress", "itemsBackedUp"); found {
		items := int(itemsBackedUp)
		progress.ItemsBackedUp = &items
	}

	if bytesBackedUp, found, _ := unstructured.NestedInt64(backup.Object, "status", "progress", "bytesBackedUp"); found {
		progress.BytesBackedUp = &bytesBackedUp
	}

	// Return nil if no progress info found
	if progress.TotalItems == nil && progress.ItemsBackedUp == nil && progress.BytesBackedUp == nil {
		return nil
	}

	return progress
}