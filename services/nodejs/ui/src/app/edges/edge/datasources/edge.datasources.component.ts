import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import { DataSource } from '../../../model/index';

import { TableBaseComponent } from '../../../base-components/table.base.component';
import { RegistryService } from '../../../services/registry.service';
import { DataSourceBaseComponent } from '../../../base-components/datasource.base.component';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

interface pmap<T> {
  [key: string]: T;
}

@Component({
  selector: 'app-edge-datasources',
  templateUrl: './edge.datasources.component.html',
  styleUrls: ['./edge.datasources.component.css'],
})
export class EdgeDataSourcesComponent extends DataSourceBaseComponent
  implements OnInit, OnDestroy {
  columns = ['Name', 'Type', 'Edge'];

  data = [];
  isConfirmLoading = false;
  sub = null;
  edgeId = null;

  datasourceDatatypeMap: pmap<string[]> = {};

  mqttTopicsMap: pmap<number> = {};

  sortMap = {
    Name: null,
    Type: null,
    Edge: null,
  };

  // to resolve the naming conflict between the table title and the key from table data source
  mapping = {
    Name: 'name',
    Type: 'type',
    Edge: 'edgeName',
  };

  constructor(
    router: Router,
    http: Http,
    private route: ActivatedRoute,
    private registryService: RegistryService
  ) {
    super(router, http);
  }

  async fetchDataSources() {
    return this.http
      .get(`/v1/edges/${this.edgeId}/datasources`, getHttpRequestOptions())
      .toPromise()
      .then(
        x => x.json(),
        rej =>
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchDataSources()
          )
      );
  }

  onClickAddDataSource() {
    this.router.navigate(
      [{ outlets: { popup: ['datasources', 'create-datasource'] } }],
      { queryParams: { edgeId: this.edgeId } }
    );
  }

  onClickUpdateTableRow() {
    const ds = this._displayData.find(d => d.id === this._rowIndex);
    console.log('>>> update, item=', ds);
    this.registryService.register(ds.id, ds);
    this.router.navigate(
      [{ outlets: { popup: ['datasources', 'create-datasource'] } }],
      { queryParams: { edgeId: this.edgeId, id: ds.id } }
    );
  }
  onClickViewTableRow() {
    const ds = this._displayData.find(d => d.id === this._rowIndex);

    this.registryService.register(ds.id, ds);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['datasources', 'create-datasource'] } }],
      { queryParams: { id: ds.id }, queryParamsHandling: 'merge' }
    );
  }

  onClickEntity(entity) {
    // this.router.navigate(['project', project._id]);
    alert('clicked data source ' + entity.name);
  }

  onClickRemoveTableRow() {
    this.isConfirmLoading = true;
    let toDeletes = [];
    if (this._allChecked) toDeletes = this._displayData.filter(x => x.checked);
    else if (this._indeterminate)
      toDeletes = this._displayData.filter(x => x.checked);
    else toDeletes = this._displayData.filter(x => x.id === this._rowIndex);
    const promises = toDeletes.map(c =>
      this.http
        .delete(`/v1/datasources/${c.id}`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.isConfirmLoading = false;
        this.fetchData();
      },
      rej => {
        handleAuthError(
          () => alert('Failed to delete datasources'),
          rej,
          this.router,
          this.http,
          () => this.onClickRemoveTableRow()
        );
        this.fetchData();
        this.isConfirmLoading = false;
      }
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

  getMqttTopics(dataSource: DataSource) {
    let mqttTopics = this.mqttTopicsMap[dataSource.id];
    if (!mqttTopics) {
      const fdtMap = {};
      dataSource.fields.forEach(f => {
        fdtMap[f.mqttTopic] = true;
      });
      mqttTopics = Object.keys(fdtMap).length;
      this.mqttTopicsMap[dataSource.id] = mqttTopics;
    }
    return mqttTopics;
  }

  ngOnInit() {
    this.sub = this.route.parent.params.subscribe(params => {
      this.edgeId = params['id'];
      this.routerEventUrl = `/edge/${this.edgeId}/datasources`;
      super.ngOnInit();
    });
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
    super.ngOnDestroy();
  }

  onFetchData() {
    const data = this.data;
    data.forEach(ds => {
      this.edges.forEach(e => {
        if (e.id === ds.edgeId) {
          ds.edgeName = e.name;
          return;
        }
      });
    });
    this.data = data;
  }
}
