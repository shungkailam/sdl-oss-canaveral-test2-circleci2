import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { RegistryService } from '../services/registry.service';
import { TableBaseComponent } from '../base-components/table.base.component';
import { AggregateInfo, Edge, DataSource, DataStream } from '../model/index';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import { OnBoardService } from '../services/onboard.service';
import { AuthService } from '../guards/auth.service';
import { dSourceDstreamMatch } from '../utils/dSourceDstreamMatchUtil';

@Component({
  selector: 'app-edges',
  templateUrl: './edges.component.html',
  styleUrls: ['./edges.component.css'],
})
export class EdgesComponent extends TableBaseComponent {
  columns = [
    'Name',
    'IP',
    'Memory',
    'CPU',
    'Storage Capacity',
    'Associated Data Sources',
  ];

  data = [];
  isConfirmLoading = false;
  isModalConfirmLoading = false;
  fetchTimer = null;
  viewModal = false;

  sortMap = {
    Name: null,
    IP: null,
    Memory: null,
    CPU: null,
    'Storage Capacity': null,
    'Associated Data Sources': null,
  };

  mapping = {
    Name: 'name',
    Ip: 'ipAddress',
    Memory: 'memory',
    CPU: 'cpu',
    'Storage Capacity': 'storage',
    'Associated Data Sources': 'dataSources',
  };

  isLoading = false;

  // subscribe to router event for create edge
  routerEventUrl = '/edges';

  isDeleteModalVisible = false;
  alertClosed = false;
  toDelete = [];

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService,
    private onboardService: OnBoardService,
    private authService: AuthService
  ) {
    super(router);
  }

  async fetchData() {
    this.isLoading = true;
    if (this.authService.isAuthenticated()) {
      if (!this.onboardService.isOnBoarded(this.authService.getUser())) {
        this.router.navigate([{ outlets: { popup: ['welcome', 'alpha'] } }], {
          queryParamsHandling: 'merge',
        });
        return;
      }
    }

    let promise = [];
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/edgesInfo', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http
        .post(
          '/v1/common/aggregates',
          {
            type: 'datasource',
            field: 'edgeId',
          },
          getHttpRequestOptions()
        )
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/datasources', getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 4) {
          const data: Edge[] = res[0].json();
          const edgesData = res[1].json();
          const aggregate: AggregateInfo[] = res[2].json();
          const dataSources: DataSource[] = res[3].json();
          data.forEach(d => {
            d['memory'] = '--';
            d['cpu'] = '--';
            d['storage'] = '--';

            const e = aggregate.find(a => a.key === d.id);
            if (e) {
              d['dataSources'] = e.doc_count;
            } else {
              d['dataSources'] = 0;
            }
            if (d['dataSources'] > 0) d['disable'] = true;

            edgesData.forEach(ed => {
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

  onClickEntity(entity) {
    this.router.navigate(['edge', entity.id], { queryParamsHandling: 'merge' });
  }

  onClickCreateEdge() {
    this.router.navigate([{ outlets: { popup: ['edges', 'create-edge'] } }], {
      queryParamsHandling: 'merge',
    });
  }

  onClickRemoveTableRow() {
    this.viewModal = false;
    this.isDeleteModalVisible = true;
    this.isConfirmLoading = true;
    this.toDelete = [];

    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDelete = this._displayData.filter(x => x.checked);
    if (this.toDelete.some(d => d.disable)) this.viewModal = true;
    this._rowIndex = '';
  }

  doDeleteEdge() {
    this.isModalConfirmLoading = true;
    const promises = this.toDelete.map(c =>
      this.http.delete(`/v1/edges/${c.id}`, getHttpRequestOptions()).toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.fetchData();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
        this.viewModal = false;
      },
      err => {
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
        this.viewModal = false;
        handleAuthError(
          () => alert('Failed to delete edge'),
          err,
          this.router,
          this.http,
          () => this.doDeleteEdge()
        );
      }
    );
  }

  onClickUpdateTableRow() {
    const edge = this._displayData.find(e => e.id === this._rowIndex);
    console.log('>>> update, item=', edge);
    this.registryService.register(edge.id, edge);
    this._rowIndex = '';

    this.router.navigate([{ outlets: { popup: ['edges', 'create-edge'] } }], {
      queryParams: { id: edge.id },
      queryParamsHandling: 'merge',
    });
  }
  onClickViewTableRow() {
    const edge = this._displayData.find(e => e.id === this._rowIndex);
    this.registryService.register(edge.id, edge);
    this._rowIndex = '';

    this.router.navigate([{ outlets: { popup: ['edges', 'create-edge'] } }], {
      queryParams: { id: edge.id },
      queryParamsHandling: 'merge',
    });
  }

  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      // return !this._displayData.some(
      //   d => d.checked && (d.associatedDataSources || d.associatedDataStreams)
      // );
      return true;
    }
    return false;
  }

  handleDeleteEdgeCancel() {
    this.isDeleteModalVisible = false;
    this.isModalConfirmLoading = false;
    this.isConfirmLoading = false;
  }

  handleDeleteEdgeOk() {
    this.doDeleteEdge();
  }

  onCloseAlert() {
    this.alertClosed = true;
  }
}
