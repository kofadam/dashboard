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
	"net/http"

	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/api/pkg/resource/customresourcedefinition"
	"k8s.io/dashboard/api/pkg/resource/dataselect"
	"k8s.io/dashboard/client"
	"k8s.io/dashboard/types"
)

// ScheduleList contains a list of Schedule resources in the cluster.
type ScheduleList struct {
	ListMeta types.ListMeta `json:"listMeta"`
	Items    []Schedule     `json:"items"`
}

// Schedule represents a Velero schedule resource
type Schedule struct {
	ObjectMeta types.ObjectMeta `json:"objectMeta"`
	TypeMeta   types.TypeMeta   `json:"typeMeta"`
}

// GetScheduleList returns a list of all Schedule resources in the cluster.
func GetScheduleList(request *http.Request, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*ScheduleList, error) {
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

	// Use dashboard's CRD framework to get Velero schedule objects
	crdList, err := customresourcedefinition.GetCustomResourceObjectList(
		apiExtClient, config, namespace, dsQuery, "schedules.velero.io")
	if err != nil {
		return nil, err
	}

	// Convert CRD items to schedule items
	var items []Schedule
	for _, item := range crdList.Items {
		schedule := Schedule{
			ObjectMeta: types.ObjectMeta{
				Name:      item.ObjectMeta.Name,
				Namespace: item.ObjectMeta.Namespace,
			},
			TypeMeta: types.TypeMeta{
				Kind: "Schedule",
			},
		}
		items = append(items, schedule)
	}

	return &ScheduleList{
		ListMeta: types.ListMeta{TotalItems: len(items)},
		Items:    items,
	}, nil
}
