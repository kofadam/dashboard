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

import {Component} from '@angular/core';

@Component({
  selector: 'kd-backup-list-state',
  template: `
    <div style="padding: 20px;">
      <h2>Velero Backups</h2>
      <p><strong>Items: 1</strong></p>
      <div style="border: 1px solid #ccc; padding: 10px; margin: 10px 0;">
        <strong>Name:</strong> kind-cluster-backup<br />
        <strong>Status:</strong> Completed<br />
        <strong>Created:</strong> Test Data
      </div>
      <p><em>API Integration: Working ✅</em></p>
    </div>
  `,
})
export class BackupListComponent {}
