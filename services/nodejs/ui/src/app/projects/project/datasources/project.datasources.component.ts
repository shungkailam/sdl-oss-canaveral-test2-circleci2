import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { DataSource } from '../../../model/index';
import { RegistryService } from '../../../services/registry.service';
import { DataSourceBaseComponent } from '../../../base-components/datasource.base.component';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

interface PMap<T> {
  [key: string]: T;
}

@Component({
  selector: 'app-project-datasources',
  templateUrl: './project.datasources.component.html',
  styleUrls: ['./project.datasources.component.css'],
})
export class ProjectDatasourcesComponent extends DataSourceBaseComponent {
  columns = ['Name', 'Type', 'Edge', 'Fields'];

  isDeleteModalVisible = false;
  isModalConfirmLoading = false;
  isConfirmLoading = false;
  datasourceDatatypeMap: PMap<string[]> = {};

  sortMap = {
    Name: null,
    Type: null,
    Edge: null,
    Fields: null,
  };

  // to resolve the naming conflict between the table title and the key from table data source

  mapping = {
    Edge: 'edgeName',
    Type: 'type',
    Name: 'name',
    Fields: 'fieldsCount',
  };

  isLoading = false;
  toDelete = [];
  queryParamSub = null;
  projectId = '';

  constructor(
    router: Router,
    http: Http,
    private route: ActivatedRoute,
    private registryService: RegistryService
  ) {
    super(router, http);
    this.queryParamSub = this.route.parent.params.subscribe(params => {
      if (params && params.id) {
        this.projectId = params.id;
        this.routerEventUrl = `/project/${this.projectId}/datasources`;
      }
    });
  }
  fetchDataSources() {
    return this.http
      .get(`v1/projects/${this.projectId}/datasources`, getHttpRequestOptions())
      .toPromise()
      .then(
        x => x.json(),
        e =>
          handleAuthError(null, e, this.router, this.http, () =>
            this.fetchDataSources()
          )
      );
  }

  onClickEntity(entity) {
    alert('clicked data source ' + entity.name);
  }

  onClickCreateDataSource() {
    this.router.navigate(
      [{ outlets: { popup: ['datasources', 'create-datasource'] } }],
      { queryParamsHandling: 'merge' }
    );
  }

  getDataType(dataSource: DataSource) {
    let dataType = this.datasourceDatatypeMap[dataSource.id];
    if (!dataType) {
      const fdtMap = {};
      dataSource.fields.forEach(f => {
        fdtMap[f.fieldType] = true;
      });
      dataType = Object.keys(fdtMap).sort();
      this.datasourceDatatypeMap[dataSource.id] = dataType;
    }
    return dataType;
  }

  onClickUpdateTableRow() {
    const ds = this._displayData.find(d => d.id === this._rowIndex);
    this.registryService.register(ds.id, ds);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['datasources', 'create-datasource'] } }],
      { queryParams: { id: ds.id }, queryParamsHandling: 'merge' }
    );
  }
  onClickViewTableRow() {
    const ds = this._displayData.find(d => d.id === this._rowIndex);
    ds.projectId = this.projectId;
    this.registryService.register(ds.id, ds);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['datasources', 'create-datasource'] } }],
      {
        queryParams: { id: ds.id, projectId: ds.projectId },
        queryParamsHandling: 'merge',
      }
    );
  }

  onFetchData() {
    const data = this.data;

    data.forEach(ds => {
      ds.fieldsCount = '';
      this.edges.forEach(e => {
        if (e.id === ds.edgeId) {
          ds.edgeName = e.name;
          return;
        }
      });
      if (ds.fields) {
        if (ds.fields.length > 2) {
          let fieldsName = '';
          for (let i = 0; i < 2; i++) {
            fieldsName += ds['fields'][i].name + ', ';
          }
          let count = ds.fields.length - 2;
          ds.fieldsCount +=
            fieldsName.substr(0, fieldsName.length - 2) +
            ' and ' +
            count +
            ' more';
        }
        if (ds.fields.length === 2 || ds.fields.length === 1) {
          ds.fields.forEach((f, i) => {
            ds.fieldsCount += f.name + ', ';
          });
          ds.fieldsCount = ds['fieldsCount'].substr(
            0,
            ds['fieldsCount'].length - 2
          );
        }
        if (ds.fields.length === 0) ds.fieldsCount = '0';
      }
    });
    this.data = data;
  }
  ngOnDestroy() {
    this.queryParamSub.unsubscribe();
    super.ngOnDestroy();
    this.unsubscribeRouterEventMaybe();
  }
  handleDeleteDatasourceCancel() {
    this.isConfirmLoading = false;
    this.isDeleteModalVisible = false;
  }
}
