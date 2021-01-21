import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { ScriptRuntime } from '../../../model/index';
import { TableBaseComponent } from '../../../base-components/table.base.component';

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
  selector: 'app-scripts-create-runtime-popup',
  templateUrl: './scripts.create-runtime.popup.component.html',
  styleUrls: ['./scripts.create-runtime.popup.component.css'],
})
export class ScriptsCreateRuntimePopupComponent extends TableBaseComponent {
  queryParamSub = null;
  name = '';
  desp = '';
  imagePath = '';
  containerRegistry = [];
  showAddContainer = false;
  containerCreationDisabled = true;
  runtimeNameFound = false;
  cloudModel = {};
  languages = ['golang', 'node', 'python'];
  language = '';
  selectedContainer = '';
  radioValue = '';
  isUpdate = false;
  originalName = '';
  runtimeId = '';
  data = [];
  dockerfile = '';
  showAddDockerFile = false;
  showMonacoEditor = false;
  builtin = false;
  globalRuntime = false;
  userCreatedRuntime = false;
  themes = [
    {
      name: 'Normal',
      value: 'vs',
    },
    {
      name: 'Dark',
      value: 'vs-dark',
    },
    {
      name: 'High Contrast Dark',
      value: 'hc-black',
    },
  ];
  theme = 'vs';
  diffObj = {
    originalValue: '',
    modifiedValue: '',
  };
  showDiff = false;
  deleteDockerModalVisible = false;
  projectId = null;
  isLoading = false;
  projectsData = [];
  context = '';
  isConfirmLoading = false;
  dockerProfiles = [];
  isCreateConfirmLoading = false;

  constructor(
    private route: ActivatedRoute,
    router: Router,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.projectId) {
        this.projectId = params.projectId;
        this.context = 'project';
      }
      if (params && params.id) {
        this.runtimeId = params.id;
        this.isUpdate = true;
        const runtime: ScriptRuntime = this.regService.get(params.id);
        if (runtime) {
          this.name = runtime.name;
          this.originalName = runtime.name;
          this.desp = runtime.description || '';
          this.language = runtime.language;
          this.imagePath = runtime.dockerRepoURI;
          this.selectedContainer = runtime.dockerProfileID;
          this.dockerfile = runtime.dockerfile;
          this.builtin = runtime.builtin;
          this.projectId = runtime.projectId;
          if (this.projectId && !this.builtin) this.userCreatedRuntime = true;
          if (!this.projectId && !this.builtin) this.globalRuntime = true;
        }
      } else {
        this.userCreatedRuntime = true;
      }
    });
  }
  fetchData() {
    this.isLoading = true;
    let promise = [];
    this.projectsData = [];
    promise.push(
      this.http.get('/v1/scriptruntimes', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http
        .get('/v1/containerregistries', getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/users', getHttpRequestOptions()).toPromise()
    );

    Promise.all(promise).then(
      response => {
        if (response.length === 4) {
          this.data = response[0].json();
          this.dockerProfiles = response[1].json();
          const projects = response[2].json();
          const users = response[3].json();
          if (this._sherlockRole !== '') {
            projects.forEach(p => {
              if (p.users) {
                p.users.forEach(pUser => {
                  users.some(u => {
                    if (u.id === pUser.userId) {
                      if (
                        u.email.trim().toLowerCase() ===
                        this._sherlockUsername.trim().toLowerCase()
                      ) {
                        this.projectsData.push(p);
                      }
                    }
                  });
                });
              }
            });
          } else this.projectsData = projects;
        }
        if (this.projectId || this.globalRuntime) {
          this.filterDockerProfiles();
        }
        this.isLoading = false;
      },
      reject => {
        handleAuthError(null, reject, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }

  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onCreateRuntime() {
    this.isConfirmLoading = true;
    this.isCreateConfirmLoading = true;
    if (this.projectId === 'global') this.projectId = '';
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    const runtime: ScriptRuntime = {
      builtin: false,
      description: this.desp || '',
      dockerProfileID: this.selectedContainer,
      dockerRepoURI: this.imagePath,
      language: this.language,
      name: this.name,
      tenantId: tenantId,
      projectId: this.projectId,
      version: 0,
    };

    if (this.dockerfile) {
      runtime['dockerfile'] = this.dockerfile;
    }

    const method = this.isUpdate ? 'put' : 'post';
    if (this.isUpdate) {
      runtime['id'] = this.runtimeId;
    }
    this.http[method]('/v1/scriptruntimes', runtime, getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          this.isConfirmLoading = false;
          this.isCreateConfirmLoading = false;
          this.router.navigate([{ outlets: { popup: null } }]);
        },
        rej => {
          const warning =
            'Failed to ' +
            (method === 'post' ? 'create' : 'update') +
            ' runtime';
          handleAuthError(
            () => alert(warning),
            rej,
            this.router,
            this.http,
            () => this.onCreateRuntime()
          );
          this.isConfirmLoading = false;
          this.isCreateConfirmLoading = false;
          this.router.navigate([{ outlets: { popup: null } }]);
        }
      );
  }

  onClickAddProfile() {
    this.showAddContainer = true;
  }

  onClickAddDockerProfile(isEdit) {
    this.showAddDockerFile = true;
    let self = this;
    setTimeout(function() {
      self.showMonacoEditor = true;
    }, 0);
    if (isEdit) {
      this.diffObj.originalValue = this.dockerfile;
    }
  }

  onUploadDockerFile() {
    this.showAddDockerFile = false;
  }

  handleFileSelect(event) {
    if (event.target.files.length && event.target.files[0].name) {
      let file = event.target.files[0];
      if (file) {
        const reader = new FileReader();
        reader.readAsText(file, 'UTF-8');
        reader.onload = evt => {
          this.dockerfile = (<any>evt.target).result;
        };
        reader.onerror = function(evt) {
          alert('Failed to read file!');
        };
      }
    } else {
      this.dockerfile = '';
    }
  }

  onChangeTheme() {
    if (window['monaco']['editor']) {
      window['monaco']['editor'].setTheme(this.theme);
    }
  }

  displayDiffEditor() {
    // set diffObj for diff editor before making it visible
    this.showDiff = !this.showDiff;
    if (this.showDiff) {
      this.diffObj.modifiedValue = this.dockerfile;
    } else {
      // set value form diff editor incase user modified it there.
      this.dockerfile = this.diffObj.modifiedValue;
    }
  }

  cloudContentChange(cloudModel) {
    this.containerCreationDisabled = cloudModel.btnDisabled;
    this.cloudModel = cloudModel.data;
    this.radioValue = cloudModel.radioValue;
  }

  disableRuntimeCreation() {
    if (this.builtin || this.globalRuntime) {
      return (
        !this.name ||
        !this.imagePath ||
        !this.language ||
        !this.selectedContainer ||
        this.runtimeNameFound
      );
    }
    return (
      !this.name ||
      !this.imagePath ||
      !this.language ||
      !this.selectedContainer ||
      this.runtimeNameFound ||
      !this.projectId
    );
  }

  onUploadContainer(isDone) {
    this.isConfirmLoading = true;
    if (!isDone) {
      this.showAddContainer = false;
      return;
    }

    this.cloudModel['tenantId'] = this.regService.get(REG_KEY_TENANT_ID);
    if (this.radioValue === 'new') {
      this.cloudModel['type'] = 'ContainerRegistry';
    }
    if (this.radioValue === 'cloud') {
      this.cloudModel['email'] = this._sherlockUsername;
    }

    this.http
      .post('/v1/containerregistries', this.cloudModel, getHttpRequestOptions())
      .toPromise()
      .then(
        r => {
          this.isConfirmLoading = false;
          this.showAddContainer = false;
          this.fetchData();
        },
        rej => {
          this.isConfirmLoading = false;
          handleAuthError(null, rej, this.router, this.http, () =>
            this.onUploadContainer(isDone)
          );
        }
      );
  }

  checkRuntimeNameDuplicate(name) {
    const n = name.toLowerCase().trim();
    this.runtimeNameFound = false;
    for (let i = 0; i < this.data.length; i++) {
      const o = this.originalName.toLowerCase().trim();
      const d = this.data[i].name.toLowerCase().trim();
      if (d === n && n !== o) {
        this.runtimeNameFound = true;
        break;
      }
    }
  }

  onClickDeleteDockerProfile() {
    this.deleteDockerModalVisible = true;
  }

  OnConfirmDeleteDockerFile(canDelete) {
    if (!canDelete) {
      this.deleteDockerModalVisible = false;
      return;
    }
    this.dockerfile = '';
    this.deleteDockerModalVisible = false;
  }

  filterDockerProfiles() {
    this.containerRegistry = [];
    if (this.projectId === 'global' || this.globalRuntime) {
      this.projectsData.forEach(pd => {
        if (pd.dockerProfileIds && pd.dockerProfileIds.length > 0) {
          pd.dockerProfileIds.forEach(dId => {
            this.dockerProfiles.forEach(dp => {
              if (dp.id === dId) {
                if (this.containerRegistry.some(cr => cr.id == dId)) {
                } else this.containerRegistry.push(dp);
              }
            });
          });
        }
      });
      return;
    }
    this.projectsData.forEach(pd => {
      if (this.projectId === pd.id) {
        if (pd.dockerProfileIds && pd.dockerProfileIds.length > 0) {
          pd.dockerProfileIds.forEach(dId => {
            this.dockerProfiles.forEach(dp => {
              if (dp.id === dId) {
                if (this.containerRegistry.some(cr => cr.id == dId)) {
                } else this.containerRegistry.push(dp);
              }
            });
          });
        }
      }
    });
  }
}
