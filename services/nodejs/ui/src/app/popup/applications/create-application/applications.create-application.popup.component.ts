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
import { Application } from '../../../model/application';
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
  selector: 'app-applications-create-application-popup',
  templateUrl: './applications.create-application.popup.component.html',
  styleUrls: ['./applications.create-application.popup.component.css'],
})
export class ApplicationsCreateApplicationPopupComponent extends TableBaseComponent {
  queryParamSub = null;
  current = 0;
  applicationNameFound = false;
  edgeSelectionVisible = false;
  edges = [];
  allEdges = [];
  selectedEdges = [];
  selectedEdgesCopy = [];
  selectAllEdges = false;
  name = '';
  desp = '';
  appId = '';
  file = null;
  code = '';
  data = [];
  isLoading = false;
  isUpdate = false;
  appl = {};
  projectsData = [];
  associatedProjectId = '';
  isConfirmLoading = false;
  noEdges = false;
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
  originalName = '';
  context = '';

  constructor(
    private route: ActivatedRoute,
    router: Router,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
  }

  async fetchData() {
    this.getProjects();
    this.getApplications();
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.projectId) {
        this.associatedProjectId = params.projectId;
        this.context = 'project';
      } else if (params && params.id) {
        this.isUpdate = true;
        const appl: Application = this.regService.get(params.id);
        console.log(appl);
        if (appl) {
          this.name = appl.name;
          this.originalName = appl.name;
          this.desp = appl.description || '';
          this.code = appl.yamlData;
          this.diffObj.originalValue = this.code;
          this.appId = appl.id;
          this.associatedProjectId = appl.projectId;
          this.http
            .get(
              `/v1/projects/${this.associatedProjectId}/edges`,
              getHttpRequestOptions()
            )
            .toPromise()
            .then(
              response => {
                this.edges = response.json();
                this.allEdges = this.edges.slice();
                if (appl.edgeIds && appl.edgeIds.length) {
                  this.selectedEdges = this.edges.filter(
                    e => appl.edgeIds.indexOf(e.id) !== -1
                  );
                } else {
                  // temporary for backward compatibility
                  this.selectedEdges = [];
                }
                this.selectedEdgesCopy = this.selectedEdges.slice();
                this.selectedEdges.forEach(ele => {
                  ele.selected = true;
                });
              },
              rej => {
                handleAuthError(null, rej, this.router, this.http, () =>
                  this.fetchData()
                );
              }
            );
        }
      }
    });
  }

  async getProjects() {
    this.projectsData = [];
    let users = [];
    let projects = [];
    await this.http
      .get('/v1/users', getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          users = response.json();
        },
        error => {
          handleAuthError(null, error, this.router, this.http, () =>
            this.getProjects()
          );
        }
      );
    await this.http
      .get('/v1/projects', getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          projects = response.json();
        },
        error => {
          handleAuthError(null, error, this.router, this.http, () =>
            this.getProjects()
          );
        }
      );
    if (this._sherlockRole !== '') {
      projects.forEach(p => {
        if (p.users) {
          p.users.forEach(pUser => {
            users.forEach(u => {
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
    } else this.projectsData = projects;
  }

  getApplications() {
    this.http
      .get('/v1/applications', getHttpRequestOptions())
      .toPromise()
      .then(
        res => {
          this.data = res.json();
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.getApplications()
          );
        }
      );
  }

  onClosePopup = function(goBack) {
    if (goBack && this.current === 1) {
      this.current = 0;
      return;
    }
    this.router.navigate([{ outlets: { popup: null } }]);
  };

  onCreateApplication() {
    if (this.current === 0) {
      this.current = 1;
      return;
    } else {
      this.isConfirmLoading = true;
      const tenantid = this.regService.get(REG_KEY_TENANT_ID);
      const edgeIds = this.selectedEdges.map(e => e.id);
      const body: Application = {
        edgeIds,
        name: this.name,
        yamlData: this.code.replace(/\t/g, '  '),
        tenantId: tenantid,
        description: this.desp || '',
        projectId: this.associatedProjectId,
      };

      if (this.appId) {
        body['id'] = this.appId;
      }

      const method = this.isUpdate ? 'put' : 'post';
      this.http[method]('/v1/application', body, getHttpRequestOptions())
        .toPromise()
        .then(
          res => {
            this.isConfirmLoading = false;
            this.router.navigate([{ outlets: { popup: null } }]);
          },
          reject => {
            this.isConfirmLoading = false;
            const warning =
              'Failed to ' +
              (method === 'post' ? 'create' : 'update') +
              ' application';
            handleAuthError(
              () => alert(warning),
              reject,
              this.router,
              this.http,
              () => this.onCreateApplication()
            );
          }
        );
    }
  }

  onOpenEdgesSelection(isEdit) {
    if (!isEdit) {
      this.http
        .get('/v1/edges', getHttpRequestOptions())
        .toPromise()
        .then(
          response => {
            this.edges = response.json();
            const edges = [];
            this.projectsData.forEach(pd => {
              if (this.associatedProjectId === pd.id)
                if (pd.edgeIds && pd.edgeIds.length > 0) {
                  pd.edgeIds.forEach(dId => {
                    this.edges.forEach(dp => {
                      if (dp.id === dId) {
                        if (edges.some(cr => cr.id == dId)) {
                        } else edges.push(dp);
                      }
                    });
                  });
                }
            });
            this.edges = edges;
            this.edges.forEach(ele => {
              ele.selected = false;
            });
            this.allEdges = this.edges.slice();
            this.selectedEdges = [];
            this.edgeSelectionVisible = true;
          },
          rej => {
            handleAuthError(null, rej, this.router, this.http, () =>
              this.onOpenEdgesSelection(isEdit)
            );
          }
        );
    } else {
      this.edgeSelectionVisible = true;
    }
  }

  onUploadEdges() {
    this.edgeSelectionVisible = false;
    this.selectedEdgesCopy = this.selectedEdges.slice();
    console.log('uploaded');
  }

  onSelectAllEdges() {
    if (this.selectAllEdges) {
      this.selectedEdges = this.edges.slice();
      this.edges.forEach(ele => {
        ele.selected = true;
      });
    } else {
      this.selectedEdges = [];
      this.edges.forEach(ele => {
        ele.selected = false;
      });
    }
  }

  removeEdge = function(edge) {
    edge.selected = false;
    this.onSelectEdge(edge);
    this.selectedEdgesCopy = this.selectedEdges.slice();
  };

  onSelectEdge = function(edge) {
    if (edge.selected) {
      this.selectedEdges.push(edge);
    } else {
      let index = -1;
      for (let i = 0; i < this.selectedEdges.length; i++) {
        if (this.selectedEdges[i].name === edge.name) {
          index = i;
          break;
        }
      }
      this.selectedEdges.splice(index, 1);
    }
    if (this.selectedEdges.length === this.allEdges.length) {
      this.selectAllEdges = true;
    } else {
      this.selectAllEdges = false;
    }
  };

  onFilterChange = function() {
    const searchVal = this.searchVal.trim().toLowerCase();
    const newEdges = [];

    this.allEdges.forEach(e => {
      const name = e.name.toLowerCase();
      if (searchVal.length === 0 || name.indexOf(searchVal) > -1) {
        newEdges.push(e);
      }
    });

    this.edges = [];
    this.selectedEdges = [];
    for (let i = 0; i < this.allEdges.length; i++) {
      for (let j = 0; j < newEdges.length; j++) {
        if (this.allEdges[i].id === newEdges[j].id) {
          this.edges.push(this.allEdges[i]);
          if (this.allEdges[i].selected) {
            this.selectedEdges.push(this.allEdges[i]);
          }
          break;
        }
      }
    }
  };

  disableBtn() {
    if (this.current === 0) {
      return (
        this.selectedEdgesCopy.length === 0 ||
        this.name.length === 0 ||
        this.applicationNameFound ||
        !this.associatedProjectId
      );
    }
    return this.code.length === 0 || this.code.length > 30720;
  }

  handleFileSelect(event) {
    if (
      event.target.files.length &&
      event.target.files[0].name &&
      (event.target.files[0].name.indexOf('.yaml') > -1 ||
        event.target.files[0].name.indexOf('.yml') > -1)
    ) {
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
      this.code = '';
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
      this.diffObj.modifiedValue = this.code;
    } else {
      // set value form diff editor incase user modified it there.
      this.code = this.diffObj.modifiedValue;
    }
  }

  checkApplicationNameDuplicate(name) {
    const n = name.toLowerCase().trim();
    this.applicationNameFound = false;
    for (let i = 0; i < this.data.length; i++) {
      const o = this.originalName.toLowerCase().trim();
      const d = this.data[i].name.toLowerCase().trim();
      if (d === n && n !== o) {
        this.applicationNameFound = true;
        break;
      }
    }
  }
  filterEdges() {
    this.projectsData.forEach(pd => {
      if (pd.id === this.associatedProjectId) {
        if (!pd.edgeIds) this.noEdges = true;
        else this.noEdges = false;
      }
    });
  }
}
