import { Component } from '@angular/core';
import {
  Router,
  ActivatedRoute,
  NavigationEnd,
  ParamMap,
} from '@angular/router';
import { Http, Headers, RequestOptions } from '@angular/http';
import { TableBaseComponent } from '../base-components/table.base.component';
import { AggregateInfo } from '../model/index';
import { RegistryService } from '../services/registry.service';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import * as uuidv4 from 'uuid/v4';
@Component({
  selector: 'app-categories',
  templateUrl: './categories.component.html',
  styleUrls: ['./categories.component.css'],
})
export class CategoriesComponent extends TableBaseComponent {
  columns = [
    'Name',
    'Values',
    'Assigned Edges',
    'Assigned Data Sources',
    'Associated Data Streams',
  ];
  data = [];
  isConfirmLoading = false;
  isModalConfirmLoading = false;
  // subscribe to router event for create category
  routerEventUrl = '/categories';

  sortMap = {
    Name: null,
    Values: null,
    'Assigned Edges': null,
    'Assigned Data Sources': null,
    'Associated Data Streams': null,
  };

  // to resolve the naming conflict between the table title and the key from table data source
  mapping = {
    Name: 'name',
    Values: 'values',
    'Assigned Edges': 'associatedEdges',
    'Assigned Data Sources': 'associatedDataSources',
    'Associated Data Streams': 'associatedDataStreams',
  };

  isLoading = false;
  isDeleteModalVisible = false;
  alertClosed = false;
  viewModal = false;
  multipleCategories = false;
  associatedCategories = [];
  toDelete = [];

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private registryService: RegistryService,
    private http: Http
  ) {
    super(router);
    this.data = [];
  }

  async fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http.get('/v1/categories', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/datasources', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/datastreams', getHttpRequestOptions()).toPromise()
    );

    Promise.all(promise).then(
      res => {
        if (res.length === 4) {
          const data = res[0].json();
          const edges = res[1].json();
          const dataSources = res[2].json();
          const dataStreams = res[3].json();
          data.forEach(d => {
            const dataStreamsInfo = dataStreams.filter(dst => {
              return (
                !!dst.originSelectors &&
                dst.originSelectors.filter(cat => {
                  return cat.id === d.id;
                }).length > 0
              );
            });

            const dataSourceInfo =
              (!!dataSources &&
                dataSources.filter(ds => {
                  return (
                    !!ds.selectors &&
                    ds.selectors.filter(cat => {
                      return cat.id === d.id;
                    }).length > 0
                  );
                })) ||
              [];

            const valuesInfo = [];
            d.values.forEach(val => {
              let datasources = [];
              let datastreams = 0;
              var edges = [];
              dataSourceInfo.forEach(ds => {
                if (ds.selectors.find(c => c.value === val)) {
                  if (!edges.includes(ds.edgeId)) edges.push(ds.edgeId);
                  datasources.push(ds.id);
                }
              });
              dataStreamsInfo.forEach(dst => {
                if (
                  !!dst.originSelectors &&
                  dst.originSelectors.find(c => c.value === val)
                ) {
                  datastreams++;
                }
              });
              valuesInfo.push({
                name: val,
                associatedEdges: edges,
                associatedDataSources: datasources,
                associatedDataStreams: datastreams,
              });
            });
            d.valuesInfo = valuesInfo;

            const edgesInfo = [];
            dataSourceInfo.forEach(ds => {
              if (!edgesInfo.includes(ds.edgeId)) edgesInfo.push(ds.edgeId);
            });

            d.associatedDataStreams = dataStreamsInfo.length;
            d.associatedDataSources = dataSourceInfo.length;
            d.associatedEdges = edgesInfo.length;
            if (
              d.associatedDataStreams > 0 ||
              d.associatedDataSources > 0 ||
              d.associatedEdges > 0
            )
              d.disable = true;
          });

          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.isLoading = false;
        }
      },
      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }

  onClickCreateCategory() {
    this.router.navigate(
      [{ outlets: { popup: ['categories', 'create-category'] } }],
      { queryParamsHandling: 'merge' }
    );
  }

  onClickRemoveTableRow() {
    this.viewModal = false;
    this._dataStreamsCount = 0;
    this._dataSourcesCount = 0;
    this.isDeleteModalVisible = true;
    this.isConfirmLoading = true;
    this.isLoading = true;
    this.associatedCategories = [];

    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDelete = this._displayData.filter(x => x.checked);

    this.toDelete.forEach(d => {
      if (d.associatedDataStreams > 0) {
        this._dataStreamsCount += d.associatedDataStreams;
        if (this.associatedCategories.length === 0)
          this.associatedCategories.push(d);
        else if (this.associatedCategories.some(ac => ac.id !== d.id))
          this.associatedCategories.push(d);
      }

      if (d.associatedDataSources > 0) {
        this._dataSourcesCount += d.associatedDataSources;
        if (this.associatedCategories.length === 0)
          this.associatedCategories.push(d);
        else if (this.associatedCategories.some(ac => ac.id !== d.id))
          this.associatedCategories.push(d);
      }
    });

    if (this.toDelete.length === 1) this.multipleCategories = false;
    else this.multipleCategories = true;

    if (this._dataStreamsCount > 0 || this._dataSourcesCount > 0)
      this.viewModal = true;
    else this.viewModal = false;

    this._rowIndex = '';
    this.isLoading = false;
  }

  doDeleteCategory() {
    this.isModalConfirmLoading = true;
    const promises = this.toDelete.map(c =>
      this.http
        .delete(`/v1/categories/${c.id}`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.fetchData();
        this.isConfirmLoading = false;
        this.isModalConfirmLoading = false;
        this.isDeleteModalVisible = false;
      },
      err => {
        this.isConfirmLoading = false;
        this.isModalConfirmLoading = false;
        this.isDeleteModalVisible = false;
        handleAuthError(
          () => alert('Failed to delete categories.'),
          err,
          this.router,
          this.http,
          () => this.doDeleteCategory()
        );
      }
    );
  }
  onClickUpdateTableRow() {
    const cat = this._displayData.find(c => c.id === this._rowIndex);
    cat.action = 'update';
    console.log('>>> update, item=', cat);
    this.registryService.register(cat.id, cat);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['categories', 'create-category'] } }],
      { queryParams: { id: cat.id }, queryParamsHandling: 'merge' }
    );
  }
  onClickViewTableRow() {
    const cat = this._displayData.find(c => c.id === this._rowIndex);
    cat.action = 'update';
    this.registryService.register(cat.id, cat);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['categories', 'create-category'] } }],
      { queryParams: { id: cat.id }, queryParamsHandling: 'merge' }
    );
  }

  onClickDuplicateTableRow() {
    const cat = this._displayData.find(c => c.id === this._rowIndex);
    cat.id = uuidv4();
    cat.action = 'duplicate';
    console.log('>>> duplicate, item=', cat);
    this.registryService.register(cat.id, cat);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['categories', 'create-category'] } }],
      { queryParams: { id: cat.id }, queryParamsHandling: 'merge' }
    );
  }
  onClickOpenValues(entity) {
    this.registryService.register(entity.id, entity);
    this.router.navigate(['category', entity.id], {
      queryParamsHandling: 'merge',
    });
  }

  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      return true;
    }
    return false;
  }

  onClickEntity(entity) {
    // this.router.navigate(['project', project._id]);
  }

  handleDeleteCategoryOk() {
    this.doDeleteCategory();
  }
  handleDeleteCategoryCancel() {
    this.isConfirmLoading = false;
    this.isModalConfirmLoading = false;
    this.isDeleteModalVisible = false;
  }
  onCloseAlert() {
    this.alertClosed = true;
  }
}
