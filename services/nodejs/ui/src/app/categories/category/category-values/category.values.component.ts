import { Component, OnInit, OnDestroy } from '@angular/core';
import {
  Router,
  ActivatedRoute,
  NavigationEnd,
  ParamMap,
} from '@angular/router';
import { Category } from '../../../model/index';
import { Http, Headers, RequestOptions } from '@angular/http';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { AggregateInfo } from '../../../model/index';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import * as uuidv4 from 'uuid/v4';

@Component({
  selector: 'app-category-values',
  templateUrl: './category.values.component.html',
  styleUrls: ['./category.values.component.css'],
})
export class CategoryValuesComponent extends TableBaseComponent {
  columns = [
    'Name',
    'Assigned Edges',
    'Assigned Data Sources',
    'Associated Data Streams',
  ];
  popupColumns = [];
  data = [];
  categoryId = null;
  categoryName = null;
  sub = null;
  isConfirmLoading = false;
  isCreateModalVisible = false;
  popupData = [];
  edgesInfo = [];

  popupSortMap = {};
  popupMapping = {};
  popupHeader = '';
  sortMap = {
    Name: null,
    'Assigned Edges': null,
    'Assigned Data Sources': null,
    'Associated Data Streams': null,
  };
  mapping = {
    Name: 'name',
    'Assigned Edges': 'associatedEdges',
    'Assigned Data Sources': 'associatedDataSources',
    'Associated Data Streams': 'associatedDataStreams',
  };

  isLoading = false;
  isDeleteModalVisible = false;
  alertClosed = false;
  edges = [];
  dataSources = [];

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private registryService: RegistryService,
    private http: Http
  ) {
    super(router);
  }

  async fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http.get('/v1/datasources', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/edgesInfo', getHttpRequestOptions()).toPromise()
    );

    Promise.all(promise).then(
      response => {
        if (response.length === 3) {
          this.dataSources = response[0].json();
          const data = response[1].json();
          this.edgesInfo = response[2].json();
          data.forEach(d => {
            d['memory'] = '';
            d['cpu'] = '';
            d['storage'] = '';

            this.edgesInfo.forEach(ed => {
              if (ed.id === d.id) {
                let memory = '',
                  TotalMemory = '',
                  storage = '',
                  totalStorage = '';

                if (
                  ed['MemoryFreeKB'] &&
                  ed['MemoryFreeKB'] !== '' &&
                  ed['TotalMemoryKB'] &&
                  ed['TotalMemoryKB'] !== ''
                )
                  memory =
                    Math.round(
                      (parseInt(ed['TotalMemoryKB']) -
                        parseInt(ed['MemoryFreeKB'])) /
                        Math.pow(1024, 2)
                    ) + ' GB of ';
                else memory = '-';

                if (ed['TotalMemoryKB'] && ed['TotalMemoryKB'] !== '')
                  TotalMemory =
                    Math.round(
                      parseInt(ed['TotalMemoryKB']) / Math.pow(1024, 2)
                    ) + ' GB';
                else TotalMemory = '-';

                if (memory === '-' && TotalMemory !== '-')
                  d['memory'] = '- of ' + TotalMemory;
                else d['memory'] = memory + TotalMemory;

                if (ed['CPUUsage'] && ed['CPUUsage'] !== '')
                  d['cpu'] = ed['CPUUsage'] + ' %';
                else d['cpu'] = '--';

                if (ed['TotalStorageKB'] && ed['TotalStorageKB'] !== '')
                  totalStorage =
                    Math.round(
                      parseInt(ed['TotalStorageKB']) / Math.pow(1024, 2)
                    ) + ' GB';
                else totalStorage = '-';

                if (
                  ed['StorageFreeKB'] &&
                  ed['StorageFreeKB'] !== '' &&
                  ed['TotalStorageKB'] &&
                  ed['TotalStorageKB'] !== ''
                )
                  storage =
                    Math.round(
                      (parseInt(ed['TotalStorageKB']) -
                        parseInt(ed['StorageFreeKB'])) /
                        Math.pow(1024, 2)
                    ) + ' GB of ';
                else storage = '-';

                if (storage === '-' && totalStorage !== '-')
                  d['storage'] = '- of ' + totalStorage;
                else d['storage'] = storage + totalStorage;
              }
            });
          });
          this.edges = data;
          this.isLoading = false;
        }
      },
      error => {
        handleAuthError(null, error, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }
  ngOnInit() {
    this.sub = this.route.parent.params.subscribe(params => {
      const category = this.registryService.get(params['id']);
      this.categoryId = category.id;
      this.data = category.valuesInfo;
      this.data.sort((a, b) => a.name.localeCompare(b.name));
      this.routerEventUrl = `/category/${this.categoryId}/category-values`;
      super.ngOnInit();
    });
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
    super.ngOnDestroy();
  }

  createEdgesPopupData(entities) {
    this.popupColumns = ['Name', 'Edge IP', 'Memory', 'CPU', 'Capacity'];
    this.popupSortMap = {
      Name: null,
      'Edge IP': null,
      Memory: null,
      CPU: null,
      Capacity: null,
    };
    this.popupMapping = {
      Name: 'name',
      'Edge IP': 'prop1',
      Memory: 'prop2',
      CPU: 'prop3',
      Capacity: 'prop4',
    };
    entities.forEach(entity => {
      const edge = this.edges.find(edge => edge.id === entity);
      this.popupData.push({
        name: edge.name,
        prop1: edge.ipAddress,
        prop2: edge.memory,
        prop3: edge.cpu,
        prop4: edge.storage,
      });
    });
  }
  createDataSourcesPopupData(entities) {
    this.popupColumns = ['Name', 'Type', 'Associated Edge', 'Protocol'];
    this.popupSortMap = {
      Name: null,
      Type: null,
      'Associated Edge': null,
      Protocol: null,
    };
    this.popupMapping = {
      Name: 'name',
      Type: 'prop1',
      'Associated Edge': 'prop2',
      Protocol: 'prop3',
    };
    entities.forEach(entity => {
      const ds = this.dataSources.find(ds => ds.id === entity);
      const edgeName = this.edges.find(e => e.id === ds.edgeId).name;
      this.popupData.push({
        name: ds.name,
        prop1: ds.type,
        prop2: edgeName,
        prop3: ds.protocol,
      });
    });
  }
  getModalTitle() {
    return this._rowIndex;
  }
  updateRowIndex(event) {
    this._rowIndex = event.currentTarget.attributes.rowValue.value;
  }
  showModal(entities, event): void {
    this.popupData = [];

    if (event.currentTarget.className === 'edges') {
      this.createEdgesPopupData(entities);
      this.popupHeader = 'Associated Edges';
    } else {
      this.createDataSourcesPopupData(entities);
      this.popupHeader = 'Associated Data Sources';
    }

    this.isCreateModalVisible = true;
  }

  handleOk(): void {
    this.isCreateModalVisible = false;
  }

  handleCancel(): void {
    this.isCreateModalVisible = false;
  }
}
