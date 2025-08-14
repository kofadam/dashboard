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
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktypes "k8s.io/apimachinery/pkg/types"

	metricapi "k8s.io/dashboard/api/pkg/integration/metric/api"
	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/api/pkg/resource/dataselect"
	"k8s.io/dashboard/types"
)

// Velero Backup GroupVersionResource for dynamic client
var BackupGVR = schema.GroupVersionResource{
	Group:    "velero.io",
	Version:  "v1",
	Resource: "backups",
}

// The code below allows to perform complex data section on []unstructured.Unstructured

type BackupCell unstructured.Unstructured

func (in BackupCell) GetProperty(name dataselect.PropertyName) dataselect.ComparableValue {
	switch name {
	case dataselect.NameProperty:
		if name, found, _ := unstructured.NestedString(in.Object, "metadata", "name"); found {
			return dataselect.StdComparableString(name)
		}
		return dataselect.StdComparableString("")
	case dataselect.CreationTimestampProperty:
		if creationTime, found, _ := unstructured.NestedString(in.Object, "metadata", "creationTimestamp"); found {
			if parsedTime, err := time.Parse(time.RFC3339, creationTime); err == nil {
				return dataselect.StdComparableTime(parsedTime)
			}
		}
		return dataselect.StdComparableTime(time.Now())
	case dataselect.NamespaceProperty:
		if namespace, found, _ := unstructured.NestedString(in.Object, "metadata", "namespace"); found {
			return dataselect.StdComparableString(namespace)
		}
		return dataselect.StdComparableString("")
	case dataselect.StatusProperty:
		if status, found, _ := unstructured.NestedString(in.Object, "status", "phase"); found {
			return dataselect.StdComparableString(status)
		}
		return dataselect.StdComparableString("Unknown")
	default:
		return nil
	}
}

func (in BackupCell) GetResourceSelector() *metricapi.ResourceSelector {
	namespace, _, _ := unstructured.NestedString(in.Object, "metadata", "namespace")
	name, _, _ := unstructured.NestedString(in.Object, "metadata", "name")
	uid, _, _ := unstructured.NestedString(in.Object, "metadata", "uid")
	
	return &metricapi.ResourceSelector{
		Namespace:    namespace,
		ResourceType: types.ResourceKindBackup,
		ResourceName: name,
		UID:          ktypes.UID(uid),
	}
}

func ToCells(std []unstructured.Unstructured) []dataselect.DataCell {
	cells := make([]dataselect.DataCell, len(std))
	for i := range std {
		cells[i] = BackupCell(std[i])
	}
	return cells
}

func FromCells(cells []dataselect.DataCell) []unstructured.Unstructured {
	std := make([]unstructured.Unstructured, len(cells))
	for i := range std {
		std[i] = unstructured.Unstructured(cells[i].(BackupCell))
	}
	return std
}

// getBackupListStatus extracts status information from backup list
func getBackupListStatus(backups []unstructured.Unstructured) common.ResourceStatus {
	info := common.ResourceStatus{}
	
	for _, backup := range backups {
		status := getBackupStatusFromSingle(&backup)
		
		switch status.Status {
		case BackupStatusFailed:
			info.Failed++
		case BackupStatusCompleted:
			info.Succeeded++
		case BackupStatusInProgress:
			info.Running++
		default:
			info.Pending++
		}
	}

	return info
}

// getBackupStatusFromSingle extracts status from a single backup
func getBackupStatusFromSingle(item *unstructured.Unstructured) BackupStatus {
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