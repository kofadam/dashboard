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

  import {HttpParams} from '@angular/common/http';
  import {ChangeDetectionStrategy, ChangeDetectorRef, Component} from '@angular/core';
  import {Observable} from 'rxjs';
  import {ResourceListBase} from '../../../../common/resources/list';
  import {NotificationsService} from '../../../../common/services/global/notifications';
  import {NamespacedResourceService} from '../../../../common/services/resource/resource';
  import {VeleroBackup, VeleroBackupList} from '../../../../common/interfaces/velero';
  import {ListGroupIdentifier, ListIdentifier} from '../../../../common/components/resourcelist/groupids';


@Component({
  selector: 'kd-backup-list',
  templateUrl: './template.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BackupListComponent extends ResourceListBase<VeleroBackupList, VeleroBackup> {
  constructor(
    private readonly backup_: NamespacedResourceService<VeleroBackupList>,
    notifications: NotificationsService,
    cdr: ChangeDetectorRef
  ) {
    super('backup', notifications, cdr);
    this.id = ListIdentifier.backup;
    this.groupId = ListGroupIdentifier.workloads;
  }

  getResourceObservable(params?: HttpParams): Observable<VeleroBackupList> {
    return this.backup_.get('backup', undefined, undefined, params);
  }

  map(backupList: VeleroBackupList): VeleroBackup[] {
    return backupList.items;
  }

  getDisplayColumns(): string[] {
    return ['name', 'created'];
  }
}
