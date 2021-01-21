import { Component, Output, EventEmitter } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

interface ParamMetadata {
  key: string;
  val: string;
}

function newParamMetadata() {
  return {
    key: uuidv4(),
    val: '',
  };
}
@Component({
  selector: 'app-components-create-container-popup',
  templateUrl: './components.popup.create-container.html',
  styleUrls: ['./components.popup.create-container.css'],
})
export class ComponentsCreateContainerPopupComponent {
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
  queryParamSub = null;
  disableOption = false;
  profileType = '';
  showPassword = false;
  showAWSUI = false;
  showGCPUI = false;
  cloudServer = '';
  cloudName = '';
  cloudModel = {};
  invalidContName = false;
  invalidCloudName = false;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private http: Http,
    private regService: RegistryService
  ) {
    this.fetchClouds();
  }

  async fetchClouds() {
    this.http
      .get('/v1/cloudcreds', getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          const data = response.json();
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
        },
        err => {
          handleAuthError(null, err, this.router, this.http, () =>
            this.fetchClouds()
          );
        }
      );
  }

  @Output() valueChange = new EventEmitter();
  valueChanged() {
    if (this.radioValue === 'new') {
      this.cloudModel = {
        radioValue: this.radioValue,
        data: {
          id: '',
          tenantId: '',
          name: this.profileName,
          description: this.profileDescp,
          pwd: this.profilePass,
          userName: this.profileUser,
          type: this.profileType,
          server: this.profileHostAddress,
          email: this.profileEmail,
          credentials: '',
        },
        btnDisabled: this.isCreateDisabled(),
      };
    } else {
      this.cloudModel = {
        radioValue: this.radioValue,
        data: {
          id: '',
          tenantId: '',
          cloudCredsID: this.cloud.id,
          name: this.cloudRegProfileName,
          description: this.cloudDescp,
          pwd: '',
          userName: '',
          type: this.profileType,
          email: '',
          server: this.cloudServer,
          credentials: '',
        },
        btnDisabled: this.isCreateDisabled(),
      };
    }
    this.valueChange.emit(this.cloudModel);
  }

  isCreateDisabled() {
    if (this.radioValue === 'new') {
      return (
        !this.profileName ||
        !this.profileHostAddress ||
        !this.profileUser ||
        !this.profilePass ||
        !this.profileEmail ||
        this.invalidContName
      );
    }
    if (this.radioValue === 'cloud') {
      return (
        !this.cloud.name ||
        !this.cloudRegProfileName ||
        !this.cloudServer ||
        this.invalidCloudName
      );
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

  checkProfileNameValid(entity) {
    let regex = '^[a-z0-9]([a-z0-9-.]*[a-z0-9])?$';
    if (entity === 'new') {
      if (!this.profileName.match(regex) && this.profileName !== '')
        this.invalidContName = true;
      else this.invalidContName = false;
    }
    if (entity === 'cloud') {
      if (
        !this.cloudRegProfileName.match(regex) &&
        this.cloudRegProfileName !== ''
      )
        this.invalidCloudName = true;
      else this.invalidCloudName = false;
    }
  }

  showPlaceHolder() {
    let placeHolder = 'Server URL';
    if (this.showAWSUI)
      return (
        placeHolder +
        ' (Ex: https://<Account number>.dkr.ecr.<region>.amazonaws.com)'
      );
    if (this.showGCPUI && this.cloud.name !== '')
      return placeHolder + ' (Ex: https://gcr.io)';
    else return placeHolder;
  }

  compareFn(c1, c2): boolean {
    return c1 && c2 ? c1.name === c2.name : c1 === c2;
  }
}
