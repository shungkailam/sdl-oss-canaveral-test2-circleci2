import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import * as uuidv4 from 'uuid/v4';
import { TableBaseComponent } from '../../../base-components/table.base.component';

@Component({
  selector: 'app-container-create-container-popup',
  templateUrl: './container.create-container.popup.component.html',
  styleUrls: ['./container.create-container.popup.component.css'],
})
export class ContainerCreateContainerPopupComponent extends TableBaseComponent {
  profile = null;
  radioValue = 'new';
  profileId = '';
  profileName = '';
  profileDescp = '';
  profileHostAddress = '';
  profileUser = '';
  profilePass = '';
  profileEmail = '';
  cloud = { name: '', id: '', type: '' };
  cloudRegProfileName = '';
  cloudDescp = '';
  cloudEmail = '';
  isLoading = false;
  isConfirmLoading = false;
  clouds = [];
  duplicateprofileNameFound = false;
  duplicateCloudRegprofileNameFound = false;
  regProfiles = [];
  queryParamSub = null;
  disableOption = false;
  profileType = '';
  showPassword = false;
  showAWSUI = false;
  showGCPUI = false;
  cloudServer = '';
  cloudName = '';
  invalidContName = false;
  invalidCloudName = false;
  longName = false;
  viewModal = false;
  cloudId = '';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.id) {
        // id param exists - update case
        let profile = this.regService.get(params.id);
        if (profile) {
          this.profileId = profile.id;
          this.profile = profile;
          this.profileName = profile.name;
          this.profileDescp = profile.description;
          this.profileUser = profile.userName;
          this.profilePass = profile.pwd;
          this.profileEmail = profile.email;
          this.profileHostAddress = profile.server;
          this.cloudRegProfileName = profile.name;
          this.cloudDescp = profile.description;
          this.cloudEmail = profile.email;
          this.cloudServer = profile.server;
          this.cloudId = profile.cloudCredsID;

          if (profile.disable) this.viewModal = true;
          else this.viewModal = false;

          if (profile.type === 'AWS' || profile.type === 'GCP')
            this.radioValue = 'cloud';
          else {
            this.radioValue = 'new';
          }
        }
      }
    });
  }
  async fetchData() {
    this.isLoading = true;
    this.clouds = [];
    let promise = [];
    promise.push(
      this.http.get('/v1/cloudcreds', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http
        .get('/v1/containerregistries', getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 2) {
          this.isLoading = false;
          const data = res[0].json();
          this.regProfiles = res[1].json();
          if (data.length === 1) {
            this.clouds = data;
            this.cloud = {
              name: this.clouds[0].name,
              id: this.clouds[0].id,
              type: this.clouds[0].type,
            };
            this.profileType = this.clouds[0].type;
            this.checkCloudType(this.cloud);
          }

          if (data.length > 1) {
            this.cloud = {
              name: 'Select..',
              id: '',
              type: '',
            };
            this.clouds.push(this.cloud);
            this.clouds.push(...data);
          }

          if (this.profileId) {
            let cl = this.clouds.find(c => c.id === this.cloudId);
            if (cl) {
              this.cloud = {
                name: cl['name'],
                id: cl['id'],
                type: cl['type'],
              };
            }
          }
        }
      },
      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchData()
        );
      }
    );
  }

  isCreateDisabled() {
    if (this.radioValue === 'new') {
      return (
        !this.profileName ||
        !this.profileHostAddress ||
        !this.profileUser ||
        !this.profilePass ||
        !this.profileEmail ||
        this.duplicateprofileNameFound ||
        this.invalidContName
      );
    }
    if (this.radioValue === 'cloud') {
      return (
        this.cloud.name === '' ||
        !this.cloudRegProfileName ||
        !this.cloudServer ||
        this.duplicateprofileNameFound ||
        this.invalidCloudName
      );
    }
  }

  onClickCreateProfile() {
    this.isConfirmLoading = true;
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    const id = '';
    let profile = {};
    if (this.radioValue === 'new') {
      this.profileType = 'ContainerRegistry';
      profile = {
        id,
        tenantId,
        name: this.profileName,
        description: this.profileDescp,
        pwd: this.profilePass,
        userName: this.profileUser,
        type: this.profileType,
        server: this.profileHostAddress,
        email: this.profileEmail,
        credentials: '',
      };
    }
    if (this.radioValue === 'cloud') {
      this.cloudEmail = this._sherlockUsername;
      profile = {
        id,
        tenantId,
        cloudCredsID: this.cloud.id,
        name: this.cloudRegProfileName,
        description: this.cloudDescp,
        pwd: '',
        userName: '',
        type: this.profileType,
        email: this.cloudEmail,
        server: this.cloudServer,
        credentials: '',
      };
    }
    let application = {
      name: '',
      yamlData: '',
      tenantId: tenantId,
      description: '',
      id: '',
    };
    let method = 'post';
    if (this.profile !== null) {
      profile['id'] = this.profile.id;
      method = 'put';
    }

    this.http[method](
      '/v1/containerregistries',
      profile,
      getHttpRequestOptions()
    )
      .toPromise()
      .then(
        r => {
          this.isConfirmLoading = false;
          this.router.navigate([{ outlets: { popup: null } }]);
        },
        err => {
          this.isConfirmLoading = false;
          const warning =
            'Failed to ' +
            (method === 'post' ? 'create' : 'update') +
            ' docker profile';
          handleAuthError(
            () => alert(warning),
            err,
            this.router,
            this.http,
            () => this.onClickCreateProfile()
          );
        }
      );
  }
  onClosePopup() {
    if (this.profileId) {
      this.regService.register(this.profileId, null);
    }
    this.router.navigate([{ outlets: { popup: null } }]);
  }
  checkProfileNameValid(entity) {
    let regex = '^[a-z0-9]([a-z0-9-.]*[a-z0-9])?$';
    if (entity === 'new') {
      if (this.profileName.length > 200) this.longName = true;
      else this.longName = false;

      if (!this.profileName.match(regex) && this.profileName !== '')
        this.invalidContName = true;
      else this.invalidContName = false;

      if (
        this.regProfiles.find(
          rp =>
            rp.name.trim().toLowerCase() ===
            this.profileName.trim().toLowerCase()
        )
      )
        this.duplicateprofileNameFound = true;
      else this.duplicateprofileNameFound = false;
    }
    if (entity === 'cloud') {
      if (this.cloudRegProfileName.length > 200) this.longName = true;
      else this.longName = false;

      if (
        !this.cloudRegProfileName.match(regex) &&
        this.cloudRegProfileName !== ''
      )
        this.invalidCloudName = true;
      else this.invalidCloudName = false;

      if (
        this.regProfiles.some(
          rp =>
            rp.name.trim().toLowerCase() ===
            this.cloudRegProfileName.trim().toLowerCase()
        )
      )
        this.duplicateCloudRegprofileNameFound = true;
      else this.duplicateCloudRegprofileNameFound = false;
    }
  }
  checkCloudType(entity) {
    this.profileType = entity.type;
    this.cloud.id = entity.id;
    this.cloud.name = entity.name;
    if (entity.type == 'AWS') {
      this.showAWSUI = true;
      this.showGCPUI = false;
    }

    if (entity.type === 'GCP') {
      this.showAWSUI = false;
      this.showGCPUI = true;
    }
  }
  showPlaceHolder() {
    let placeHolder = 'Server URL';
    if (this.showAWSUI)
      return (
        placeHolder + ' (Ex: <Account number>.dkr.ecr.<region>.amazonaws.com)'
      );
    if (this.showGCPUI && this.cloud.name !== '')
      return placeHolder + ' (Ex: https://gcr.io)';
    else return placeHolder;
  }
  compareFn(c1, c2): boolean {
    return c1 && c2 ? c1.name === c2.name : c1 === c2;
  }
}
