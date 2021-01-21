import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../services/registry.service';
import { TableBaseComponent } from '../base-components/table.base.component';
import * as uuidv4 from 'uuid/v4';
import {
  CloudCreds,
  AWSCredential,
  GCPCredential,
  CloudType,
} from '../model/index';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';

@Component({
  selector: 'app-clouds',
  templateUrl: './clouds.component.html',
  styleUrls: ['./clouds.component.css'],
})
export class CloudsComponent extends TableBaseComponent {
  columns = ['Name', 'Cloud Type', 'Description'];
  data = [];
  isConfirmLoading = false;
  isCreateModalVisible = false;
  cloudType = null;
  cloudTypes = ['select', 'AWS', 'GCP'];
  cloudProfileName = '';
  cloudProfileDescription = '';
  duplicateProfileNameFound = false;
  viewModal = false;
  multipleClouds = false;
  associatedProfiles = [];
  toDelete = [];
  isModalConfirmLoading = false;

  // AWS
  awsAccessKeyInput = '';
  awsSecretInput = '';

  // GCP - json of service account info
  gcpSvcAcctInfo = '';
  notJson = false;
  cloudToUpdate: CloudCreds = null;

  isLoading = false;
  isDeleteModalVisible = false;
  routerEventUrl = '/clouds';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
  }
  async fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http.get('/v1/cloudcreds', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http
        .get('/v1/containerregistries', getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );

    Promise.all(promise).then(
      res => {
        if (res.length === 3) {
          const data = res[0].json();
          const dockerProfilesData = res[1].json();
          const projectsData = res[2].json();
          data.forEach(p => {
            p.storageStatus = 'OK';
            p.computeStatus = 'OK';
            if (dockerProfilesData.some(dp => dp.cloudCredsID === p.id))
              p.disable = true;
            projectsData.forEach(p => {
              if (p.cloudCredentialIds) {
                if (p.cloudCredentialIds.some(pId => pId === p.id))
                  p.disable = true;
              }
            });
          });
          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.isLoading = false;
        }
      },

      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }

  onClickEntity(entity) {
    // no op
  }

  onClickCreateCloud() {
    this.isCreateModalVisible = true;
    // reset input
    this.cloudType = CloudType.SELECT;
    this.cloudProfileName = '';
    this.cloudProfileDescription = '';
    this.awsAccessKeyInput = '';
    this.awsSecretInput = '';
    this.gcpSvcAcctInfo = '';
  }

  updateRowIndex(id, option) {
    this._rowIndex = id;
    if (option.label === 'Edit') this.onClickUpdateTableRow();
    if (option.label === 'Remove') this.onClickRemoveTableRow();
    if (option.label === 'View') this.onClickViewTableRow();
  }

  onClickRemoveTableRow() {
    this.toDelete = [];
    this.associatedProfiles = [];
    this.isDeleteModalVisible = true;
    this.isConfirmLoading = true;
    this.isLoading = true;
    this.viewModal = false;

    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDelete = this._displayData.filter(x => x.checked);

    if (this.toDelete.length === 1) this.multipleClouds = false;
    else this.multipleClouds = true;

    this.toDelete.forEach(d => {
      if (d.disable) {
        this.viewModal = true;
        this.associatedProfiles.push(d);
      }
    });

    this.isLoading = false;
  }

  doDeleteCloud() {
    this.isModalConfirmLoading = true;
    const promises = this.toDelete.map(c =>
      this.http
        .delete(`/v1/cloudcreds/${c.id}`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.fetchData();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
      },
      err => {
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
        handleAuthError(
          () => alert('Failed to delete cloud'),
          err,
          this.router,
          this.http,
          () => this.doDeleteCloud()
        );
      }
    );
  }

  onClickUpdateTableRow() {
    const cloud: CloudCreds = this._displayData.find(
      cl => cl.id === this._rowIndex
    );
    if (cloud['disable']) this.viewModal = true;
    else this.viewModal = false;

    console.log('>>> update, item=', cloud);
    this.cloudToUpdate = cloud;
    this.isCreateModalVisible = true;
    // reset input
    this.cloudType = cloud.type;
    this.cloudProfileName = cloud.name;
    this.cloudProfileDescription = cloud.description;
    if (cloud.type === 'AWS') {
      this.awsAccessKeyInput = cloud.awsCredential.accessKey;
      this.awsSecretInput = cloud.awsCredential.secret;
    } else {
      this.gcpSvcAcctInfo = JSON.stringify(cloud.gcpCredential);
    }
  }
  onClickViewTableRow() {
    const cloud: CloudCreds = this._displayData.find(
      cl => cl.id === this._rowIndex
    );
    if (cloud['disable'] || this._sherlockRole === '') this.viewModal = true;
    else this.viewModal = false;

    this.cloudToUpdate = cloud;
    this.isCreateModalVisible = true;
    // reset input
    this.cloudType = cloud.type;
    this.cloudProfileName = cloud.name;
    this.cloudProfileDescription = cloud.description;
    if (cloud.type === 'AWS') {
      this.awsAccessKeyInput = cloud.awsCredential.accessKey;
      this.awsSecretInput = cloud.awsCredential.secret;
    } else {
      this.gcpSvcAcctInfo = JSON.stringify(cloud.gcpCredential);
    }
  }

  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      return true;
    }
    return false;
  }

  handleCreateCloudOk() {
    this.isModalConfirmLoading = true;
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    let id = uuidv4();
    let method = 'post';
    let awsCredential: AWSCredential = null;
    let gcpCredential: GCPCredential = null;
    if (this.cloudToUpdate) {
      id = this.cloudToUpdate.id;
      method = 'put';
      this.cloudToUpdate = null;
    }
    if (this.cloudType === 'AWS') {
      awsCredential = {
        accessKey: this.awsAccessKeyInput,
        secret: this.awsSecretInput,
      };
    } else {
      gcpCredential = JSON.parse(this.gcpSvcAcctInfo);
    }
    const cloud: CloudCreds = {
      name: this.cloudProfileName,
      description: this.cloudProfileDescription,
      type: this.cloudType,
      awsCredential,
      gcpCredential,
      id,
      tenantId,
    };
    this.http[method]('/v1/cloudcreds', cloud, getHttpRequestOptions())
      .toPromise()
      .then(
        r => {
          this.fetchData();
          this.viewModal = false;
          this.isModalConfirmLoading = false;
          this.isCreateModalVisible = false;
        },
        err => {
          this.isModalConfirmLoading = false;
          this.isCreateModalVisible = false;
          const warning =
            'Failed to ' + (method === 'post' ? 'create' : 'update') + ' cloud';
          handleAuthError(
            () => alert(warning),
            err,
            this.router,
            this.http,
            () => this.handleCreateCloudOk()
          );
        }
      );
  }
  handleCreateCloudCancel() {
    this.isCreateModalVisible = false;
    this.cloudToUpdate = null;
    this.viewModal = false;
  }

  getPopupTitle(): string {
    if (this.cloudToUpdate) {
      return 'Update Cloud Profile';
    } else {
      return 'Create Cloud Profile';
    }
  }

  onChangeCloudType() {}

  checkDuplicates() {
    if (
      this.data.some(
        c =>
          c.name.trim().toLowerCase() ===
          this.cloudProfileName.trim().toLowerCase()
      )
    )
      this.duplicateProfileNameFound = true;
    else this.duplicateProfileNameFound = false;
  }

  handleDeleteCloudOk() {
    this.isModalConfirmLoading = true;
    this.doDeleteCloud();
    this.duplicateProfileNameFound = false;
  }
  handleDeleteCloudCancel() {
    this.isConfirmLoading = false;
    this.isDeleteModalVisible = false;
    this.duplicateProfileNameFound = false;
  }
  isDisabled() {
    if (this.cloudType === 'select') {
      return true;
    }
    if (this.cloudType === 'AWS') {
      return (
        this.cloudType === 'select' ||
        !this.cloudProfileName ||
        this.duplicateProfileNameFound ||
        !this.awsAccessKeyInput ||
        !this.awsSecretInput
      );
    }
    if (this.cloudType === 'GCP') {
      return (
        this.cloudType === 'select' ||
        !this.cloudProfileName ||
        this.duplicateProfileNameFound ||
        !this.gcpSvcAcctInfo ||
        this.notJson
      );
    }
  }

  changeGcpJson() {
    try {
      const gcpJson = JSON.parse(this.gcpSvcAcctInfo);
      this.notJson = false;
    } catch (e) {
      this.notJson = true;
    }
  }
}
