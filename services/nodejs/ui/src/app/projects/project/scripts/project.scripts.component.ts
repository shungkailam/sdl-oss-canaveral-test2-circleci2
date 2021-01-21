import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

@Component({
  selector: 'app-project-scripts',
  templateUrl: './project.scripts.component.html',
  styleUrls: ['./project.scripts.component.css'],
})
export class ProjectScriptsComponent extends TableBaseComponent {
  columns = [
    'Name',
    'Language',
    'Runtime Environment',
    'Associated Data stream',
    'Last Modified',
  ];
  data = [];

  queryParamSub = null;
  projectId = '';
  routerEventUrl = '';
  isConfirmLoading = false;
  isDeleteModalVisible = false;
  alertClosed = false;
  _dataStreamsCount = 0;
  _dataSourcesCount = 0;
  datasources = [];
  datastreams = [];
  multipleScripts = false;
  projectName = '';
  toDelete = [];
  isModalConfirmLoading = false;

  sortMap = {
    Name: null,
    Language: null,
    'Runtime Environment': null,
    'Associated Data stream': null,
    'Last Modified': null,
  };

  mapping = {
    Name: 'name',
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
    this.queryParamSub = this.route.parent.params.subscribe(params => {
      if (params && params.id) {
        this.projectId = params.id;
        // id param exists - update case
        let project = this.regService.get(params.id);
        if (project) {
          this.projectName = project.name;
        }
        this.routerEventUrl = `/project/${this.projectId}/scripts`;
      }
    });
  }

  async fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http
        .get(
          `v1/projects/${this.projectId}/datasources`,
          getHttpRequestOptions()
        )
        .toPromise()
    );
    promise.push(
      this.http
        .get(
          `v1/projects/${this.projectId}/scriptruntimes`,
          getHttpRequestOptions()
        )
        .toPromise()
    );
    promise.push(
      this.http
        .get(`v1/projects/${this.projectId}/scripts`, getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http
        .get(
          `v1/projects/${this.projectId}/datastreams`,
          getHttpRequestOptions()
        )
        .toPromise()
    );
    Promise.all(promise).then(
      response => {
        if (response.length === 4) {
          this.datasources = response[0].json();
          const runtime = response[1].json();
          const data = response[2].json();
          this.datastreams = response[3].json();
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
      queryParams: { projectId: this.projectId },
      queryParamsHandling: 'merge',
    });
  }

  onClickUpdateTableRow() {
    const script = this._displayData.find(s => s.id === this._rowIndex);
    this.regService.register(script.id, script);
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
  ngOnInit() {
    super.ngOnInit();
  }
  ngOnDestroy() {
    this.queryParamSub.unsubscribe();
    super.ngOnDestroy();
    this.unsubscribeRouterEventMaybe();
  }
}
