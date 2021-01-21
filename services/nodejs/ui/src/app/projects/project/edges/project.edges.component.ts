import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { RegistryService } from '../../../services/registry.service';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { AggregateInfo, Edge, DataSource } from '../../../model/index';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { OnBoardService } from '../../../services/onboard.service';
import { AuthService } from '../../../guards/auth.service';

@Component({
  selector: 'app-project-edges',
  templateUrl: './project.edges.component.html',
  styleUrls: ['./project.edges.component.css'],
})
export class ProjectEdgesComponent extends TableBaseComponent {
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
  fetchTimer = null;
  projectId = '';
  queryParamSub = null;
  toDelete = [];
  isModalConfirmLoading = false;
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

  isDeleteModalVisible = false;
  alertClosed = false;

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService,
    private onboardService: OnBoardService,
    private authService: AuthService
  ) {
    super(router);
    this.queryParamSub = this.route.parent.params.subscribe(params => {
      if (params && params.id) {
        this.projectId = params.id;
        this.routerEventUrl = `/project/${this.projectId}/edges`;
      }
    });
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
      this.http
        .get(`v1/projects/${this.projectId}/edges`, getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('v1/edgesInfo', getHttpRequestOptions()).toPromise()
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
      this.http
        .get(
          `v1/projects/${this.projectId}/datasources`,
          getHttpRequestOptions()
        )
        .toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 4) {
          const data: Edge[] = res[0].json();
          const edgesData = res[1].json();
          const aggregate: AggregateInfo[] = res[2].json();
          const dataSources: DataSource[] = res[3].json();
          data.forEach(d => {
            d['memory'] = '';
            d['cpu'] = '';
            d['storage'] = '';

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
    edge.projectId = this.projectId;
    this.registryService.register(edge.id, edge);
    this._rowIndex = '';

    this.router.navigate([{ outlets: { popup: ['edges', 'create-edge'] } }], {
      queryParams: { id: edge.id, projectId: edge.projectId },
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
  onCloseAlert() {
    this.alertClosed = true;
  }

  ngOnDestroy() {
    this.queryParamSub.unsubscribe();
    super.ngOnDestroy();
    this.unsubscribeRouterEventMaybe();
  }
}
