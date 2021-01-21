import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { AggregateInfo } from '../../../model/index';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { Edge } from '../../../model/index';
import { reject } from 'q';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { JSONP_ERR_WRONG_RESPONSE_TYPE } from '@angular/common/http/src/jsonp';
import * as uuidv4 from 'uuid/v4';
import { element } from '../../../../../node_modules/protractor';

@Component({
  selector: 'app-application-deployment',
  templateUrl: './deployment.component.html',
  styleUrls: ['./deployment.component.css'],
})
export class ApplicationDeploymentComponent extends TableBaseComponent {
  isLoading = false;
  isDeleteModalVisible = false;
  toDelete = [];
  data = [];
  columns = [
    'Edge',
    'Status',
    'Alerts',
    'CPU Usage',
    'Memory Usage',
    'Latest Log Bundle',
  ];
  isContainerModalVisible = false;
  sub = null;
  app = null;
  appId = '';
  isConfirmLoading = false;
  contName = '';
  containersData = [];
  contTitle = '';
  logsCollected = true;
  routerEventUrl = '/applications/application/deployment';
  relatedEdges = [];
  selectedEdges = [];
  fetchLogsTimer = null;
  successLogs = {};
  clickedEdges = [];
  isLogsModalVisible = false;
  downloadLogs = false;

  appName = '';
  logsData = {};
  logs = [];

  modalCols = [
    'Container Name',
    'Container Image',
    'Status',
    'Restarts',
    'Uptime',
    'CPU Usage',
    'Memory Usage',
  ];

  sortMap = {
    Edge: null,
    Status: null,
    Alerts: null,
    'CPU USage': null,
    'Memory Usage': null,
    'Latest Log Bundle': null,
  };

  mapping = {
    Edge: 'name',
    Status: 'status',
    Alerts: 'alerts',
    'CPU Usage': 'cpu',
    'Memory Usage': 'memory',
    'Latest Log Bundle': 'time',
  };

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService
  ) {
    super(router);

    this.sub = this.route.parent.params.subscribe(params => {
      this.appId = params.id;
      this.routerEventUrl = `/application/${this.appId}/deployment`;
      this.app = this.registryService.get(params['id']);
      if (this.app) {
      } else {
        this.appId = params['id'];
      }
    });
    this.data = [];
  }

  async fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http
        .get('/v1/applicationstatus', getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/edgesInfo', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/applications', getHttpRequestOptions()).toPromise()
    );

    Promise.all(promise).then(
      res => {
        if (res.length === 4) {
          const statusData = res[0].json();
          const edge = res[1].json();
          const edgesMoreData = res[2].json();
          const appData = res[3].json();
          const edgesData = [];
          appData.some(app => {
            if (app.id === this.appId) {
              this.app = app;
            }
          });
          edge.forEach(e => {
            if (this.app && this.app.edgeIds) {
              const appEntry = this.app.edgeIds.find(ae => ae === e.id);
              if (appEntry) {
                edgesData.push(e);
              }
            }
          });
          edgesData.forEach(e => {
            e['alerts'] = '--';
            e['memory'] = '--';
            e['cpu'] = '--';
            edgesMoreData.some(ed => {
              if (ed.id === e.id) {
                let memory = '',
                  TotalMemory = '';
                if (ed['MemoryFreeKB'] && ed['MemoryFreeKB'] !== '')
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
                  e['memory'] = '- of ' + TotalMemory;
                else e['memory'] = memory + TotalMemory;

                if (ed['CPUUsage'] && ed['CPUUsage'] !== '')
                  e['cpu'] = ed['CPUUsage'] + ' %';
                else e['cpu'] = '--';
              }
            });
          });

          let status = '';
          let uptime = '';
          let podName = '';
          let edgeId = '';
          statusData.forEach(s => {
            if (s.applicationId === this.appId) {
              edgesData.forEach(e => {
                if (e.id === s.edgeId) {
                  let cRunning = 0;
                  let cTotal = 0;
                  let containersData = [];
                  if (s.appStatus.podStatusList !== null) {
                    s.appStatus.podStatusList.forEach(p => {
                      if (
                        p.status.containerStatuses &&
                        p.status.containerStatuses !== null
                      ) {
                        p.status.containerStatuses.forEach(c => {
                          let cpu = '-';
                          let memory = '-';
                          let today = new Date();
                          let date1 = new Date(today);
                          cTotal++;
                          if (c.state.running && c.state.running !== null) {
                            status = 'Running';
                            let started = new Date(c.state.running.startedAt);
                            // The number of milliseconds in one day
                            cRunning++;
                            let diff = Math.abs(
                              date1.getTime() - started.getTime()
                            );
                            let days = Math.ceil(diff / (1000 * 3600 * 24));
                            if (days <= 1) uptime = days + ' Day';
                            else uptime = days + ' Days';
                          } else if (
                            c.state.waiting &&
                            c.state.waiting !== null
                          ) {
                            status = 'Waiting';
                            uptime = '-';
                          } else if (
                            c.state.terminated &&
                            c.state.terminated !== null
                          ) {
                            status = 'Terminated';
                            let started = new Date(
                              c.state.terminated.startedAt
                            );
                            // The number of milliseconds in one day
                            cRunning++;
                            let diff = Math.abs(
                              date1.getTime() - started.getTime()
                            );
                            let days = Math.ceil(diff / (1000 * 3600 * 24));
                            if (days <= 1) uptime = days + ' Day';
                            else uptime = days + ' Days';
                          } else {
                            status = 'Unknown';
                            uptime = '-';
                          }
                          if (s.appStatus.podMetricsList !== null) {
                            s.appStatus.podMetricsList.forEach(pm => {
                              pm.containers.forEach(pc => {
                                if (
                                  pc.name.trim().toLowerCase() ===
                                  c.name.trim().toLowerCase()
                                ) {
                                  cpu = pc.usage.cpu;
                                  memory = pc.usage.memory;
                                }
                              });
                            });
                          }
                          let cont = {
                            name: c.name,
                            yaml: c.image,
                            status: status,
                            restarts: c.restartCount,
                            uptime: uptime,
                            memory: memory,
                            cpu: cpu,
                          };
                          if (
                            edgeId !== e.id ||
                            podName.trim().toLowerCase() !==
                              p.metadata.name.trim().toLowerCase()
                          ) {
                            containersData.push(cont);
                          }
                        });
                      }
                      podName = p.metadata.name;
                    });
                  }
                  e['status'] = cRunning + ' of ' + cTotal + ' Running';
                  e['contData'] = containersData;
                  edgeId = e.id;
                }
              });
            }
          });
          edgesData.sort((a, b) => a.name.localeCompare(b.name));
          this.data = edgesData;
          this.fetchLogs().then(v => {
            this.isLoading = false;
          });
        } else {
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
  async fetchLogs() {
    try {
      this.logsData = {};
      this.successLogs = {};
      this.logs = await this.http
        .get('/v1/logs/entries', getHttpRequestOptions())
        .toPromise()
        .then(response => response.json());

      this.logs = this.logs.sort(function(a, b) {
        let date1 = new Date(a.createdAt);
        let date2 = new Date(b.createdAt);
        return date1 > date2 ? -1 : date1 < date2 ? 1 : 0;
      });

      this.logs.forEach((l, i) => {
        if (!this.logsData[l.edgeId]) this.logsData[l.edgeId] = [l];
        else this.logsData[l.edgeId].push(l);
        if (l['status'] === 'SUCCESS') {
          if (!this.successLogs[l.edgeId]) this.successLogs[l.edgeId] = [l];
          else this.successLogs[l.edgeId].push(l);
        }
      });

      this.data.forEach((e, i) => {
        if (this.logsData[e.id]) {
          if (this.logsData[e.id][0]['status'] === 'SUCCESS') {
            let time = new Date(
              this.logsData[e.id][0].createdAt
            ).toLocaleString();
            e['time'] = time;
            e['collectingLogs'] = false;
            e['disable'] = false;
            e['logsCollected'] = true;
            clearInterval(this.fetchLogsTimer);
          } else if (this.logsData[e.id][0]['status'] === 'PENDING') {
            console.log('logs pending');
            e['collectingLogs'] = true;
            e['disable'] = true;
            if (this.successLogs[e.id]) {
              e['logsCollected'] = true;
              let time = new Date(
                this.successLogs[e.id][0].createdAt
              ).toLocaleString();
              e['time'] = time;
            }
          } else {
            e['collectingLogs'] = false;
            if (this.successLogs[e.id]) {
              let time = new Date(
                this.successLogs[e.id][0].createdAt
              ).toLocaleString();
              e['time'] = time;
              e['disable'] = false;
            } else {
              e['time'] = '--';
              e['disable'] = true;
            }
            if (this.selectedEdges.length > 1) {
              if (i === this.data.length - 1)
                alert(
                  'Failed to collect Logs: ' +
                    this.logsData[e.id][0]['errorMessage']
                );
            } else {
              if (this.selectedEdges.find(se => se.id === e.id))
                alert(
                  'Failed to collect Logs: ' +
                    this.logsData[e.id][0]['errorMessage']
                );
            }
            clearInterval(this.fetchLogsTimer);
          }
        } else {
          e['time'] = '--';
          e['disable'] = true;
          e['logsCollected'] = false;
        }
      });
      const edges = this.data.sort((a, b) => a.name.localeCompare(b.name));
      this.data = edges;
      this._refreshStatus();
    } catch (e) {
      handleAuthError(null, e, this.router, this.http, () => this.fetchLogs());
      this.isLoading = false;
    }
  }

  onClickEditApp() {
    this.registryService.register(this.appId, this.app);
    this.router.navigate(
      [{ outlets: { popup: ['applications', 'create-application'] } }],
      {
        queryParams: { id: this.appId },
        queryParamsHandling: 'merge',
      }
    );
  }

  isShowingEditButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      return false;
    }
    return true;
  }
  isShowingDownloadButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      return true;
    }
    return false;
  }
  onClickOpenContainers(entity) {
    if (this.data.some(e => e.id === entity.id))
      if (entity.contData) this.containersData = entity.contData;
      else this.containersData = [];

    this.contName = entity.name;
    this.contTitle = this.app.name + ': ' + this.contName;
    this.isContainerModalVisible = true;
  }
  handleContainerCancel() {
    this.isContainerModalVisible = false;
  }
  handleLogsModalCancel() {
    this.downloadLogs = false;
    this.isLogsModalVisible = false;
  }
  handleLogsDownload() {
    var link = document.createElement('a');

    link.setAttribute('download', null);
    link.style.display = 'none';

    document.body.appendChild(link);

    for (var i = 0; i < this.selectedEdges.length; i++) {
      link.setAttribute('href', this.selectedEdges[i].url);
      link.click();
    }

    document.body.removeChild(link);
    this.downloadLogs = true;
    this.isLogsModalVisible = false;
  }
  ngOnDestroy() {
    this.sub.unsubscribe();
    super.ngOnDestroy();
    clearInterval(this.fetchLogsTimer);
  }

  onClickDownloadLogs() {
    this.selectedEdges = [];
    if (this._rowIndex) {
      this.selectedEdges = this._displayData.filter(
        x => x.id === this._rowIndex
      );
    } else this.selectedEdges = this._displayData.filter(x => x.checked);

    let promises = [];
    this.selectedEdges.forEach(se => {
      if (this.successLogs[se.id]) {
        promises.push(
          this.http
            .post(
              '/v1/logs/requestDownload',
              {
                location: this.logsData[se.id][0].location,
              },
              getHttpRequestOptions()
            )
            .toPromise()
        );
      }
    });
    Promise.all(promises).then(
      response => {
        this.selectedEdges.forEach((se, i) => {
          se['url'] = response[i]._body;
        });
        this.isLogsModalVisible = true;
      },
      rej => {
        handleAuthError(
          () => alert('Failed to download logs'),
          rej,
          this.router,
          this.http,
          () => this.onClickDownloadLogs()
        );
      }
    );

    this._rowIndex = '';
  }

  uploadLogs() {
    this.selectedEdges = [];
    if (this._rowIndex) {
      this.selectedEdges = this._displayData.filter(
        x => x.id === this._rowIndex
      );
    } else this.selectedEdges = this._displayData.filter(x => x.checked);

    if (this.selectedEdges.length === 0) {
      //this.isCreateLogVisible = false;
      return;
    }
    const edgeIds = [];
    this.selectedEdges.forEach(e => {
      e['collectingLogs'] = true;
      edgeIds.push(e.id);
    });
    try {
      this.http
        .post(
          '/v1/logs/requestUpload',
          {
            edgeIds: edgeIds,
            applicationId: this.appId,
          },
          getHttpRequestOptions()
        )
        .toPromise()
        .then(
          response => {
            this.fetchLogs();
            this.startFetchingLogs();
          },
          reject => {
            this.fetchLogs();
          }
        );
    } catch (e) {}
    this._rowIndex = '';
  }
  startFetchingLogs() {
    this.fetchLogsTimer = setInterval(() => this.fetchLogs(), 60000);
  }
  _checkAll(value) {
    if (value) {
      this._displayData.forEach(data => {
        data.checked = true;
        if (data.disable) this.logsCollected = false;
      });
    } else {
      this._displayData.forEach(data => {
        data.checked = false;
        this.logsCollected = true;
      });
    }
    this._refreshStatus();
  }
  _refreshStatus() {
    const allChecked = this._displayData.every(value => value.checked === true);
    const allUnChecked = this._displayData.every(value => !value.checked);
    this._allChecked = allChecked;
    this._indeterminate = !allChecked && !allUnChecked;
    if (this._displayData.some(c => c.disable && c.checked))
      this.logsCollected = false;
    else this.logsCollected = true;
  }
}
