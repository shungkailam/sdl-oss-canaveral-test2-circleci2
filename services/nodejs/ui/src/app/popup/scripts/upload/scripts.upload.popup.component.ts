import { Component, ViewChild } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { Location } from '@angular/common';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { Script, ScriptParam } from '../../../model/index';
import { handleAuthError } from '../../../utils/authUtil';
import { lang } from 'moment';
import { TableBaseComponent } from '../../../base-components/table.base.component';

interface ParamMetadata {
  key: string;
  name: string;
  type: string;
}

function newParamMetadata() {
  return {
    key: uuidv4(),
    name: '',
    type: '',
  };
}

@Component({
  selector: 'app-scripts-upload-popup',
  templateUrl: './scripts.upload.popup.component.html',
  styleUrls: ['./scripts.upload.popup.component.css'],
})
export class ScriptsUploadPopupComponent extends TableBaseComponent {
  queryParamSub = null;
  langaugeList = ['golang', 'node', 'python'];
  language = '';
  envName = '';
  isVisible = true;
  isConfirmLoading = false;
  radioValue = 'transformation';
  scriptValue = 'upload';
  file = null;
  scriptName = '';
  code = '';
  script = null;
  desp = '';
  paramType = ['string', 'number'];
  isUpdate = false;
  projectsData = [];
  scriptAction = '';

  @ViewChild('monacoEditor') monacoEditor;
  @ViewChild('monacoDiffEditor') monacoDiffEditor;

  // which step are we on in the create flow
  current = 0;
  canDiff = false;
  showDiff = false;
  inline = false;
  diffObj = {
    originalValue: '',
    modifiedValue: '',
  };
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
  showAddParamTable = false;
  data: ParamMetadata[] = [];

  editRow: string = null;
  tempEditObject: any = {};
  duplicateScriptNameFound = false;
  scripts = [];
  allEnvironment = [];
  environmentList = [];
  environmentMap = {};
  associatedWithDs = false;
  associatedDsList = [];
  isCloned = false;
  selectAllDataStreams = false;
  selectedDS = [];
  showWarningmodel = false;
  allowDsChange = true;
  projectId = null;
  isLoading = false;
  context = '';

  constructor(
    private route: ActivatedRoute,
    router: Router,
    private location: Location,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.isLoading = true;
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.projectId) {
        this.projectId = params.projectId;
        this.context = 'project';
      } else if (params && params.id) {
        // id param exists - update case
        const script = this.regService.get(params.id);
        if (script) {
          this.isUpdate = true;
          this.script = script;
          this.language = script.language;
          this.scriptName = script.name;
          this.code = script.code;
          this.envName = script.environment;
          this.scriptAction = script.action;
          this.data = this.initParamData(script.params);
          this.showAddParamTable = this.data.length !== 0;
          this.associatedWithDs = script.associatedDs > 0 ? true : false;
          this.associatedDsList = script.associatedDsList;
          this.isCloned = script.isCloned;
          this.desp = script.description;
          this.projectId = script.projectId;
          if (params.current) {
            try {
              this.current = parseInt(params.current, 10);
            } catch (e) {
              // ignore
            }
          }
          if (this.scriptAction === 'duplicate') this.scriptName = '';
        }
      }
    });
  }

  fetchData() {
    this.isLoading = true;
    let promise = [];
    this.environmentMap = {};
    this.projectsData = [];
    promise.push(
      this.http.get('/v1/scripts', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    if (this.projectId) {
      promise.push(
        this.http
          .get(
            `/v1/projects/${this.projectId}/scriptruntimes`,
            getHttpRequestOptions()
          )
          .toPromise()
      );
    } else {
      promise.push(
        this.http.get('/v1/scriptruntimes', getHttpRequestOptions()).toPromise()
      );
      promise.push(
        this.http.get('/v1/users', getHttpRequestOptions()).toPromise()
      );
    }

    Promise.all(promise).then(
      response => {
        this.scripts = response[0].json();
        if (this.projectId) {
          this.projectsData = response[1].json();
          const allEnvironment = response[2].json();
          allEnvironment.forEach(a => {
            const language = a.language.toLowerCase();
            if (!this.environmentMap[language]) {
              this.environmentMap[language] = [];
            }
            this.environmentMap[language].push(a);
          });

          if (this.language) {
            this.onChangeLanguage(this.language);
          }
        } else {
          const projects = response[1].json();
          this.allEnvironment = response[2].json();
          const users = response[3].json();
          projects.forEach(p => {
            if (p.users) {
              p.users.forEach(pUser => {
                users.some(u => {
                  if (u.id === pUser.userId) {
                    if (
                      u.email.trim().toLowerCase() ===
                      this._sherlockUsername.trim().toLowerCase()
                    )
                      this.projectsData.push(p);
                  }
                });
              });
            }
          });
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

  changeProjectId(projectId) {
    const pid = projectId;
    const allEnvironment = this.allEnvironment.filter(
      al => !al.projectId || al.projectId === pid
    );
    this.environmentMap = {};
    allEnvironment.forEach(a => {
      const language = a.language.toLowerCase();
      if (!this.environmentMap[language]) {
        this.environmentMap[language] = [];
      }
      this.environmentMap[language].push(a);
    });
    this.language = 'golang';
    this.onChangeLanguage(this.language);
  }

  handleCreate() {
    if (this.current === 2) {
      this.showWarningmodel = true;
      return;
    }
    this.isConfirmLoading = true;
    // since we now split the workflow in two steps,
    // this method can only be called after first step
    // has been validated, so no need to check params
    // already checked in step 1
    this.createScript(this.code);
  }

  private getParams() {
    return this.data.map(({ name, type }) => ({
      name,
      type,
    }));
  }
  private initParamData(params: ScriptParam[]): ParamMetadata[] {
    return params.map(({ name, type }) => ({
      name,
      type,
      key: uuidv4(),
    }));
  }

  createScript(code) {
    this.isConfirmLoading = true;
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    const type = this.radioValue === 'lambda' ? 'Function' : 'Transformation';
    const language = this.language;
    const id = uuidv4();
    const description = this.desp;
    const env = this.environmentList.find(env => env.name === this.envName);
    const runtimeId = env.id;
    const environment = env.dockerRepoURI;
    const scriptObj: Script = {
      id,
      tenantId,
      code,
      language,
      environment,
      type,
      description,
      name: this.scriptName,
      params: this.getParams(),
      runtimeId: runtimeId,
      projectId: this.projectId,
      builtin: false,
    };
    let method = 'post';
    if (!this.isCloned && this.script) {
      scriptObj.id = this.script.id;
      method = 'put';
    }

    // now save the script
    this.http[method]('/v1/scripts', scriptObj, getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          if (
            this.isCloned &&
            this.allowDsChange &&
            this.selectedDS.length > 0
          ) {
            const promise = [];
            const newScriptId = x.json()._id;
            this.selectedDS.forEach(ds => {
              // if the current script is used by a datastream, the datastream will use the new script
              ds.transformationArgsList.forEach(arg => {
                if (arg.transformationId === this.script.id) {
                  arg.transformationId = newScriptId;
                  promise.push(
                    this.http
                      .put('/v1/datastreams', ds, getHttpRequestOptions())
                      .toPromise()
                  );
                }
              });
            });
            Promise.all(promise).then(
              res => {
                this.isVisible = false;
                this.isConfirmLoading = false;
                // must delay this, otherwise table scrolling will be broken
                setTimeout(() => this.onClosePopup());
              },
              rej => {
                this.isConfirmLoading = false;
                handleAuthError(null, rej, this.router, this.http, () =>
                  this.createScript(code)
                );
              }
            );
          } else {
            this.isVisible = false;
            this.isConfirmLoading = false;
            // must delay this, otherwise table scrolling will be broken
            setTimeout(() => this.onClosePopup());
          }
        },
        e => {
          const text = this.isCloned
            ? 'clone'
            : method === 'post' ? 'create' : 'update';
          const warning = 'Failed to ' + text + ' script';
          this.isConfirmLoading = false;
          handleAuthError(() => alert(warning), e, this.router, this.http, () =>
            this.createScript(code)
          );
        }
      );
  }

  handleCancel() {
    this.isVisible = false;
    this.onClosePopup();
  }

  onClosePopup() {
    // this could be called over another popup
    // e.g., in DataStream create popup,
    // so just use location.back to pop,
    // don't use router navigate to popup=null outlet
    this.location.back();
  }

  handleFileSelect(event) {
    if (event.target.files.length) {
      this.file = event.target.files[0];
      if (this.file) {
        const reader = new FileReader();
        reader.readAsText(this.file, 'UTF-8');
        reader.onload = evt => {
          this.code = (<any>evt.target).result;
        };
        reader.onerror = function(evt) {
          alert('Failed to read file!');
        };
      }
    } else {
      this.file = null;
    }
  }

  isCreateDisabled() {
    return !this.code || this.code.length > 30720;
  }

  onChangeLanguage(language) {
    try {
      const l = language.toLowerCase();
      this.environmentList = this.environmentMap[l];
      if (this.isUpdate) this.envName = this.script.environment;
      else this.envName = this.environmentList[0].name;
    } catch (e) {}
  }

  onBack() {
    this.current -= 1;
  }

  onNext() {
    this.detectParamChange();
    this.current += 1;
  }

  detectParamChange() {
    if (this.isCloned && this.associatedWithDs && this.current === 1) {
      const originalParams = this.initParamData(this.script.params);
      const currParams = this.getParams();
      this.allowDsChange = true;
      if (originalParams.length !== currParams.length) {
        this.allowDsChange = false;
      } else {
        for (let i = 0; i < currParams.length; i++) {
          if (
            currParams[i].name !== originalParams[i].name ||
            currParams[i].type !== originalParams[i].type
          ) {
            this.allowDsChange = false;
            break;
          }
        }
      }
    }
  }

  showNext() {
    return (
      this.current === 0 ||
      (this.current === 1 && this.isCloned && this.associatedWithDs)
    );
  }

  showBack() {
    return this.current !== 0;
  }

  // whether Next button should be disabled or not
  isNextDisabled() {
    if (this.current === 0) {
      return (
        !this.scriptName ||
        !this.envName ||
        !this.language ||
        !this.projectId ||
        this.duplicateScriptNameFound
      );
    }
    if (this.current === 1) {
      return !this.code;
    }
    return true;
  }

  /**
   * Show diff handler
   * Set the diff model object and show diff editor
   */
  disableDiff() {
    return !this.showDiff && !this.isUpdate;
  }

  displayDiffEditor() {
    // set diffObj for diff editor before making it visible
    this.showDiff = !this.showDiff;
    if (this.showDiff) {
      this.diffObj.originalValue =
        this.script && this.script.code ? this.script.code : '';
      this.diffObj.modifiedValue = this.code;
    } else {
      // set value form diff editor incase user modified it there.
      this.code = this.diffObj.modifiedValue;
    }
  }

  onChangeTheme() {
    if (window['monaco']['editor']) {
      window['monaco']['editor'].setTheme(this.theme);
    }
  }

  onClickAddParam() {
    this.showAddParamTable = true;
    if (this.data.length === 0) {
      this.onAddParam();
    }
  }

  onAddParam() {
    if (this.data[0] && (!this.data[0].name || !this.data[0].type)) {
      return;
    }
    const pm = newParamMetadata();
    this.data.unshift(pm);
    this.tempEditObject[pm.key] = pm;
    this.editRow = pm.key;
  }

  edit(data) {
    this.tempEditObject[data.key] = { ...data };
    this.editRow = data.key;
  }

  save(event, data) {
    event.stopPropagation();
    Object.assign(data, this.tempEditObject[data.key]);
    this.editRow = null;
  }

  cancel(event, data) {
    event.stopPropagation();
    this.tempEditObject[data.key] = {};
    this.editRow = null;
    const idx = this.data.findIndex(v => v.key === data.key);
    if (idx !== -1) {
      this.data.splice(idx, 1);
      if (this.data.length === 0) {
        this.showAddParamTable = false;
      }
    }
  }

  clickRow(data) {
    if (this.editRow !== data.key) {
      this.edit(data);
    }
  }

  checkDuplicateScripts(value) {
    if (
      this.scripts.some(
        sc => sc.name.trim().toLowerCase() === value.trim().toLowerCase()
      )
    )
      this.duplicateScriptNameFound = true;
    else this.duplicateScriptNameFound = false;
  }

  showLayer() {
    return this.associatedWithDs && !this.isCloned;
  }

  onSelectAllDataStreams() {
    if (this.selectAllDataStreams) {
      this.selectedDS = this.associatedDsList.slice();
      this.associatedDsList.forEach(ele => {
        ele.selected = true;
      });
    } else {
      this.selectedDS = [];
      this.associatedDsList.forEach(ele => {
        ele.selected = false;
      });
    }
  }

  onSelectDataStream(ds) {
    if (ds.selected) {
      this.selectedDS.push(ds);
    } else {
      const index = this.selectedDS.findIndex(ele => ele.id === ds.id);
      this.selectedDS.splice(index, 1);
    }

    if (this.associatedDsList.length === this.selectedDS.length) {
      this.selectAllDataStreams = true;
    } else {
      this.selectAllDataStreams = false;
    }
  }

  onConfirmClone(isConfirmed) {
    if (!isConfirmed) {
      this.showWarningmodel = false;
      return;
    } else {
      this.showWarningmodel = false;
      this.createScript(this.code);
    }
  }
}
