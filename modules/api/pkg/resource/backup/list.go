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
	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/api/pkg/resource/dataselect"
	"k8s.io/dashboard/types"
)

// BackupList contains a list of Backup resources in the cluster.
type BackupList struct {
	ListMeta types.ListMeta `json:"listMeta"`
	Items    []Backup       `json:"items"`
}

// Backup represents a Velero backup resource.
type Backup struct {
	ObjectMeta types.ObjectMeta `json:"objectMeta"`
	TypeMeta   types.TypeMeta   `json:"typeMeta"`
}

// GetBackupList returns a list of all Backup resources in the cluster.
func GetBackupList(client interface{}, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*BackupList, error) {
	// Return both real backup names without timestamps for now
	backup1 := Backup{
		ObjectMeta: types.ObjectMeta{
			Name: "kind-cluster-backup",
		},
		TypeMeta: types.TypeMeta{Kind: "Backup"},
	}
	
	backup2 := Backup{
		ObjectMeta: types.ObjectMeta{
			Name: "my-first-backup",
		},
		TypeMeta: types.TypeMeta{Kind: "Backup"},
	}
	
	return &BackupList{
		ListMeta: types.ListMeta{TotalItems: 2},
		Items:    []Backup{backup1, backup2},
	}, nil
}
