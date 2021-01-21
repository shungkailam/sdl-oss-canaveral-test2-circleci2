import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../base-components/table.base.component';
import { AggregateInfo } from '../model';
import { RegistryService } from '../services/registry.service';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';

@Component({
  selector: 'app-scripts',
  templateUrl: './scripts.component.html',
  styleUrls: ['./scripts.component.css'],
})
export class ScriptsComponent extends TableBaseComponent {
  columns = [
    'Name',
    'Project',
    'Language',
    'Runtime Environment',
    'Associated Data stream',
    'Last Modified',
  ];
  data = [];

  routerEventUrl = '/scripts';

  isConfirmLoading = false;
  isDeleteModalVisible = false;
  alertClosed = false;
  _dataStreamsCount = 0;
  _dataSourcesCount = 0;
  datasources = [];
  datastreams = [];
  multipleScripts = false;
  toDelete = [];
  projects = [];
  isModalConfirmLoading = false;

  sortMap = {
    Name: null,
    Project: null,
    Language: null,
    'Runtime Environment': null,
    'Associated Data stream': null,
    'Last Modified': null,
  };

  mapping = {
    Name: 'name',
    Project: 'project',
    Language: 'language',
    'Runtime Environment': 'environment',
    'Associated Data stream': 'associatedDs',
    'Last Modified': 'lastModified',
  };

  isLoading = false;

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
    let scripts = [];
    promise.push(
      this.http.get('/v1/datasources', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/scriptruntimes', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/scripts', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/datastreams', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/users', getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      response => {
        if (response.length === 6) {
          this.datasources = response[0].json();
          const runtime = response[1].json();
          const data = response[2].json();
          this.datastreams = response[3].json();
          const projects = response[4].json();
          const users = response[5].json();
          const currentUser = users.find(
            u => u.email.trim() === this._sherlockUsername
          );
          if (currentUser) {
            projects.forEach(p => {
              if (p.users && p.users.find(pu => pu.userId === currentUser.id)) {
                this.projects.push(p);
              }
            });
          }
          data.forEach(dd => {
            dd.associatedDs = 0;
            dd.associatedDsList = [];
            this.datastreams.forEach(dst => {
              if (
                dst.transformationArgsList.some(
                  s => s.transformationId === dd.id
                )
              ) {
                dd.associatedDs++;
                dd.associatedDsList.push(dst);
              }
            });
            let runtimeItem = runtime.find(
              r => r.dockerRepoURI === dd.environment
            );
            if (runtimeItem) dd.environment = runtimeItem.name;
            const date = new Date(dd.updatedAt);
            const time = date.toLocaleString();
            dd.lastModified = time;
            if (dd.associatedDs > 0) dd.disable = true;
            this.projects.forEach(p => {
              if (p.id === dd.projectId) dd.project = p.name;
            });
          });

          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.isLoading = false;
        }
      },
      reject => {
        handleAuthError(null, reject, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }

  onClickEntity(entity) {
    this.regService.register(entity.id, entity);
    this.router.navigate([{ outlets: { popup2: ['scripts', 'upload'] } }], {
      queryParams: { id: entity.id, current: 1 },
      queryParamsHandling: 'merge',
    });
  }

  onClickUploadScript() {
    this.router.navigate([{ outlets: { popup2: ['scripts', 'upload'] } }], {
      queryParamsHandling: 'merge',
    });
  }

  onClickUpdateTableRow() {
    const script = this._displayData.find(s => s.id === this._rowIndex);
    this.regService.register(script.id, script);
    this._rowIndex = '';
    this.router.navigate([{ outlets: { popup2: ['scripts', 'upload'] } }], {
      queryParams: { id: script.id },
      queryParamsHandling: 'merge',
    });
  }

  onClickRemoveTableRow() {
    this.isConfirmLoading = true;
    this._dataStreamsCount = 0;
    this._dataSourcesCount = 0;
    this.multipleScripts = true;
    this.toDelete = [];

    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDelete = this._displayData.filter(x => x.checked);

    this.toDelete.forEach(dd => {
      this.datastreams.forEach(dst => {
        if (
          dst.transformationArgsList.some(s => s.transformationId === dd.id)
        ) {
          this._dataStreamsCount++;
          this.datasources.forEach(ds => {
            ds.selectors.forEach(c => {
              if (
                !!dst.originSelectors &&
                dst.originSelectors.some(dstc => dstc.id === c.id)
              ) {
                this._dataSourcesCount++;
              }
            });
          });
        }
      });
    });

    this._rowIndex = '';
    this.isDeleteModalVisible = true;
  }

  onClickDuplicateTableRow() {
    let script = this._displayData.find(s => s.id === this._rowIndex);
    script.action = 'duplicate';
    script.isCloned = true;
    this.regService.register(script.id, script);
    this._rowIndex = '';
    this.router.navigate([{ outlets: { popup2: ['scripts', 'upload'] } }], {
      queryParams: { id: script.id },
      queryParamsHandling: 'merge',
    });
  }

  deleteScript() {
    const promises = this.toDelete.map(c =>
      this.http
        .delete(`/v1/scripts/${c.id}`, getHttpRequestOptions())
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
        this.isConfirmLoading = false;
        this.isModalConfirmLoading = false;
        this.isDeleteModalVisible = false;
        handleAuthError(
          () => alert('Failed to delete script'),
          err,
          this.router,
          this.http,
          () => this.deleteScript()
        );
      }
    );
  }

  handleDeleteScriptOk() {
    this.isModalConfirmLoading = true;
    this.deleteScript();
  }
  handleDeleteScriptCancel() {
    this.isConfirmLoading = false;
    this.isDeleteModalVisible = false;
  }
  onCloseAlert() {
    this.alertClosed = true;
  }
}
