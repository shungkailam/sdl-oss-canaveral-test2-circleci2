import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../services/registry.service';
import { TableBaseComponent } from '../../base-components/table.base.component';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';
import { datasourceMatchOriginSelectors } from '../../utils/modelUtil';
import { resolve4 } from 'dns';

@Component({
  selector: 'app-projects-list',
  templateUrl: './list.component.html',
  styleUrls: ['./list.component.css'],
})
export class ProjectsListComponent extends TableBaseComponent {
  columns = ['Name', 'Users', 'Edges', 'Cloud Profiles', 'Container Profiles'];
  data = [];
  isConfirmLoading = false;
  isLoading = false;
  viewModal = false;
  associatedProjects = [];
  multipleProjects = false;
  isDeleteModalVisible = false;
  isModalConfirmLoading = false;
  users = [];
  toDelete = [];
  sortMap = {
    Name: null,
    Users: null,
    Edges: null,
    'Cloud Profiles': null,
    'Container Profiles': null,
  };

  mapping = {
    Name: 'name',
    Users: 'usersDetails',
    Edges: 'edgeNumber',
    'Cloud Profiles': 'cloudShowingName',
    'Container Profiles': 'containerShowingName',
  };

  routerEventUrl = '/projects/list';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService
  ) {
    super(router);
  }
  async fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/applications', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/datastreams', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/scripts', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/scriptruntimes', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/cloudcreds', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http
        .get('/v1/containerregistries', getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/datasources', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/users', getHttpRequestOptions()).toPromise()
    );

    Promise.all(promise).then(
      response => {
        if (response.length === 9) {
          const data = response[0].json();
          const applications = response[1].json();
          const datastreams = response[2].json();
          const scripts = response[3].json();
          const runtimes = response[4].json();
          const clouds = response[5].json();
          const profiles = response[6].json();
          const datasources = response[7].json();
          this.users = response[8].json();
          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.data.forEach(p => {
            if (applications.some(a => a.projectId === p.id)) p.disable = true;
            if (datastreams.some(ds => ds.projectId === p.id)) p.disable = true;
            if (scripts.some(s => s.projectId === p.id)) p.disable = true;
            if (runtimes.some(r => r.projectId === p.id)) p.disable = true;

            if (p.users) p.usersDetails = p.users;
            else p.usersDetails = [];

            p.usersDetails.forEach(ud => {
              this.users.forEach(u => {
                if (u.id === ud.userId) {
                  if (
                    u.email.trim().toLowerCase() ===
                    this._sherlockUsername.trim().toLowerCase()
                  )
                    p.associatedUser = true;
                }
              });
            });

            p.cloudNames = [];
            if (p.cloudCredentialIds && p.cloudCredentialIds.length > 0) {
              p.cloudCredentialIds.forEach(id => {
                let cloudItems = clouds.find(ac => ac.id === id);
                if (cloudItems) {
                  p.cloudNames.push(cloudItems.name);
                }
              });
              p.cloudShowingName = this.showNames(p.cloudNames, 'clouds');
            }
            p.containerNames = [];
            if (p.dockerProfileIds && p.dockerProfileIds.length > 0) {
              p.dockerProfileIds.forEach(id => {
                let dockerItems = profiles.find(ac => ac.id === id);
                if (dockerItems) {
                  p.containerNames.push(dockerItems.name);
                }
              });
              p.containerShowingName = this.showNames(
                p.containerNames,
                'containers'
              );
            }
            if (p.edgeSelectorType === 'Explicit') {
              if (p.edgeIds) {
                p.edgeNumber = p.edgeIds.length;
              } else {
                p.edgeNumber = 0;
              }
            } else {
              const dss = datasources.filter(ds =>
                datasourceMatchOriginSelectors(ds, p.edgeSelectors, '')
              );
              p.edgeNumber = 0;
              if (dss.length) {
                let edgeMap = {};
                dss.forEach(ds => {
                  if (!edgeMap[ds.edgeId]) {
                    edgeMap[ds.edgeId] = true;
                    p.edgeNumber++;
                  }
                });
              }
            }
          });
          const finalData = data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = finalData;
        }
        this.isLoading = false;
      },
      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }

  showNames = function(arr, type) {
    if (arr.length === 0) {
      return '-';
    }
    if (arr.length === 1) {
      return arr[0];
    }
    if (arr.length === 2) {
      return arr[0] + ', ' + arr[1];
    }
    return (
      arr[0] + ', ' + arr[1] + ' and ' + (arr.length - 2) + ' other ' + type
    );
  };

  getCloudNames() {
    this.data.forEach(f => {
      f.cloudNames = [];
      if (f.cloudCredentialIds && f.cloudCredentialIds.length > 0) {
        this.http
          .get(`/v1/cloudcreds`, getHttpRequestOptions())
          .toPromise()
          .then(
            res => {
              const allClouds = res.json();
              f.cloudCredentialIds.forEach(id => {
                let cloudItems = allClouds.find(ac => ac.id === id);
                if (cloudItems) {
                  f.cloudNames.push(cloudItems.name);
                }
              });
              f.cloudShowingName = this.showNames(f.cloudNames, 'clouds');
            },
            rej => {
              handleAuthError(null, rej, this.router, this.http, () =>
                this.getCloudNames()
              );
            }
          );
      }
    });
  }

  getContainerNames() {
    this.data.forEach(f => {
      f.containerNames = [];
      if (f.dockerProfileIds && f.dockerProfileIds.length > 0) {
        this.http
          .get(`/v1/containerregistries`, getHttpRequestOptions())
          .toPromise()
          .then(
            res => {
              const allContainers = res.json();
              f.dockerProfileIds.forEach(id => {
                let dockerItems = allContainers.find(ac => ac.id === id);
                if (dockerItems) {
                  f.containerNames.push(dockerItems.name);
                }
              });
              f.containerShowingName = this.showNames(
                f.containerNames,
                'containers'
              );
            },
            rej => {
              handleAuthError(null, rej, this.router, this.http, () =>
                this.getContainerNames()
              );
            }
          );
      }
    });
  }

  getEdgeNumber() {
    this.data.forEach(d => {
      if (d.edgeSelectorType === 'Explicit') {
        if (d.edgeIds) {
          d.edgeNumber = d.edgeIds.length;
        } else {
          d.edgeNumber = 0;
        }
      } else {
        var datasources = [];
        this.http
          .get('/v1/datasources', getHttpRequestOptions())
          .toPromise()
          .then(
            cs => {
              datasources = cs.json();
              const dss = datasources.filter(ds =>
                datasourceMatchOriginSelectors(ds, d.edgeSelectors, '')
              );
              d.edgeNumber = 0;
              if (dss.length) {
                let edgeMap = {};
                dss.forEach(ds => {
                  if (!edgeMap[ds.edgeId]) {
                    edgeMap[ds.edgeId] = true;
                    d.edgeNumber++;
                  }
                });
              }
            },
            rej => {
              handleAuthError(null, rej, this.router, this.http, () =>
                this.getEdgeNumber()
              );
            }
          );
      }
    });
  }

  onClickOpenProjectDetails(project) {
    this.registryService.register(project.id, project);
    let path = '';
    if (this._sherlockRole === '') path = 'applications';
    else path = 'edges';
    this.router.navigate(['project', project.id, path]);
  }

  onClickCreateProject() {
    this.router.navigate(
      [{ outlets: { popup: ['projects', 'create-project'] } }],
      { queryParamsHandling: 'merge' }
    );
  }

  onClickRemoveTableRow() {
    this.isConfirmLoading = true;
    this.isDeleteModalVisible = true;
    this.toDelete = [];
    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDelete = this._displayData.filter(x => x.checked);

    this._rowIndex = '';
  }
  onClickViewTableRow() {
    const project = this._displayData.find(c => c.id === this._rowIndex);
    this.registryService.register(project.id, project);
    this.router.navigate(
      [{ outlets: { popup: ['projects', 'create-project'] } }],
      { queryParams: { id: project.id }, queryParamsHandling: 'merge' }
    );
  }

  onClickUpdateTableRow() {
    const project = this._displayData.find(c => c.id === this._rowIndex);
    console.log('>>> update, item=', project);
    this.registryService.register(project.id, project);
    this.router.navigate(
      [{ outlets: { popup: ['projects', 'create-project'] } }],
      { queryParams: { id: project.id }, queryParamsHandling: 'merge' }
    );
  }

  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      return !this._displayData.some(
        d => d.checked && (d.associatedDataSources || d.associatedDataStreams)
      );
    }
    return false;
  }

  handleDeleteProjectOk() {
    this.isModalConfirmLoading = true;
    let deleteList = [];
    this.toDelete.forEach(d => {
      deleteList.push(
        this.http
          .delete(`/v1/projects/${d.id}`, getHttpRequestOptions())
          .toPromise()
      );
    });
    Promise.all(deleteList).then(
      res => {
        this.fetchData();
        this.isModalConfirmLoading = true;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
      },
      rej => {
        handleAuthError(
          () => alert('Failed to delete project'),
          rej,
          this.router,
          this.http,
          () => this.handleDeleteProjectOk()
        );
        this.isModalConfirmLoading = true;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
      }
    );
  }

  handleDeleteProjectCancel() {
    this.isConfirmLoading = false;
    this.isModalConfirmLoading = false;
    this.isDeleteModalVisible = false;
  }
}
