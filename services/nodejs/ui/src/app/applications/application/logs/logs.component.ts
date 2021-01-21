import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { AggregateInfo } from '../../../model/index';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { Edge } from '../../../model/index';

@Component({
  selector: 'app-application-logs',
  templateUrl: './logs.component.html',
  styleUrls: ['./logs.component.css'],
})
export class ApplicationLogsComponent extends TableBaseComponent {
  sub = null;
  appId = '';
  searchVal = '';
  isLoading = false;
  columns = ['Name', 'Time', 'Collected for', 'Status'];
  data = [];
  isCreateLogVisible = false;
  isDeleteLogVisible = false;
  edges = [];
  allEdges = [];
  selectedEdges = [];
  selectAllEdges = false;
  entries = [];
  isDownload = false;
  relatedEdges = [];
  deleteItems = [];
  uploaded = false;
  isUploading = false;
  fetchLogsTimer = null;
  isPopupLoading = false;
  isConfirmLoading = false;
  isModalConfirmLoading = false;
  connectedEdges = [];
  allConnectedEdges = [];
  sortMap = {
    Name: null,
    Time: null,
    'Collected for': null,
    Status: null,
  };

  mapping = {
    Name: 'name',
    Time: 'time',
    'Collected for': 'count',
    Status: 'status',
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
      this.routerEventUrl = `/application/${this.appId}/logs`;
    });
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
    super.ngOnDestroy();
    clearInterval(this.fetchLogsTimer);
  }
  async fetchData() {
    this.getLogs();
    this.fetchEdges();
  }
  getLogs = function() {
    this.isLoading = true;
    let promises = [];
    promises.push(
      this.http.get('/v1/logs/entries', getHttpRequestOptions()).toPromise()
    );
    promises.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    Promise.all(promises).then(
      res => {
        if (res.length === 2) {
          const entries = res[0]
            .json()
            .filter(e => e.tags[0] && e.tags[0].value === this.appId);
          const edges = res[1].json();
          if (entries) {
            this.entries = entries.sort(function(a, b) {
              let date1 = new Date(a.createdAt);
              let date2 = new Date(b.createdAt);
              return date1 > date2 ? -1 : date1 < date2 ? 1 : 0;
            });
            let map = {};
            this.data = [];
            entries.forEach(e => {
              const date = new Date(e.createdAt);
              const time = date.toLocaleString();
              const edgeItem = edges.find(eg => eg.id === e.edgeId);
              let status = '';

              if (!map[e.batchId]) {
                const suffix = e.batchId.split('-');
                map[e.batchId] = {
                  count: 1,
                  batchId: e.batchId,
                  name:
                    'Log_' +
                    (suffix.length > 0 ? suffix[suffix.length - 1] : ''),
                  date: date,
                  time: time,
                  statusMap: {
                    PENDING: false,
                    FAILED: false,
                    SUCCESS: false,
                  },
                  edges: [],
                };
                map[e.batchId].status = this.setLogStatus(
                  map[e.batchId].statusMap,
                  e.status
                );
              } else {
                map[e.batchId].count++;
                map[e.batchId].status = this.setLogStatus(
                  map[e.batchId].statusMap,
                  e.status
                );
              }
              if (edgeItem) {
                map[e.batchId].edges.push(edgeItem);
              }
            });
            const data = [];
            for (let key in map) {
              data.push(map[key]);
            }
            data.sort((a, b) => a.time.localeCompare(b.name));
            this.data = data;
          }

          this.isLoading = false;
        }
      },

      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.getLogs()
        );
        this.isLoading = false;
      }
    );
  };

  setLogStatus(statusMap, status) {
    statusMap[status] = true;
    if (statusMap['PENDING']) return 'Collecting';
    if (statusMap['SUCCESS'] && statusMap['FAILED']) {
      return 'Partial Failure';
    }
    if (statusMap['SUCCESS']) return 'Success';
    if (statusMap['FAILED']) {
      return 'Failed';
    }
  }

  async fetchEdges() {
    let promise = [];
    this.connectedEdges = [];
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http
        .get(`/v1/application/${this.appId}`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promise).then(
      response => {
        if (response.length === 2) {
          const edges = response[0].json();
          const app = response[1].json();
          const appEdges = [];
          edges.forEach(e => {
            if (app && app.edgeIds) {
              const appEntry = app.edgeIds.find(ae => ae === e.id);
              if (appEntry) {
                appEdges.push(e);
              }
            }
          });
          this.edges = appEdges;
          this.edges.forEach(ele => {
            if (ele.connected) this.connectedEdges.push(ele);
            ele.selected = false;
          });
          this.allEdges = this.edges.slice();
          this.allConnectedEdges = this.connectedEdges.slice();
          this.selectedEdges = [];
          this.selectAllEdges = false;
          this.searchVal = '';
          this.uploaded = false;
          this.isPopupLoading = false;
        }
      },
      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchEdges()
        );
      }
    );
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

  onClickCreateLog = function() {
    this.isPopupLoading = true;
    this.isCreateLogVisible = true;
    this.fetchEdges();
    this.isDownload = false;
  };

  OnClickBundleDownload = function(log) {
    this.relatedEdges = [];
    this.searchVal = '';
    this.isPopupLoading = true;
    this.entries.forEach((e, index) => {
      if (e.batchId === log.batchId) {
        if (e.status === 'SUCCESS') {
          let promises = [];
          promises.push(
            this.http
              .get(`/v1/edges/${e.edgeId}`, getHttpRequestOptions())
              .toPromise()
          );

          promises.push(
            this.http
              .post(
                '/v1/logs/requestDownload',
                {
                  location: e.location,
                },
                getHttpRequestOptions()
              )
              .toPromise()
          );

          Promise.all(promises).then(
            res => {
              let r = res[0].json();
              if (r) {
                e.name = r.name;
              }
              r = res[1]._body;
              if (r) {
                e.url = r;
              }
              this.isPopupLoading = false;
              this.isDownload = true;
              this.isCreateLogVisible = true;
              this.relatedEdges.push(e);
            },
            rej => {
              handleAuthError(
                () => alert('Failed to download logs'),
                rej,
                this.router,
                this.http,
                () => this.OnClickBundleDownload(log)
              );
            }
          );
        }
        if (e.status === 'FAILED') {
          this.http
            .get(`/v1/edges/${e.edgeId}`, getHttpRequestOptions())
            .toPromise()
            .then(
              res => {
                this.isCreateLogVisible = true;
                this.isPopupLoading = false;
                let r = res.json();
                this.isDownload = true;
                if (r) {
                  e.name = r.name;
                }
                this.relatedEdges.push(e);
              },
              rej => {
                handleAuthError(
                  () => alert('Failed to download logs'),
                  rej,
                  this.router,
                  this.http,
                  () => this.OnClickBundleDownload(log)
                );
              }
            );
        }
      }
    });
  };

  onClickDeleteTableRow(obj) {
    this.isDeleteLogVisible = true;
    this.isConfirmLoading = true;
    this.deleteItems = [];
    if (obj) {
      this.deleteItems.push(obj);
      return;
    }
    if (this._rowIndex) {
      this.deleteItems = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.deleteItems = this._displayData.filter(x => x.checked);
    this._rowIndex = '';
  }

  OnClickDeleteBundle = function(isDelete) {
    this.isModalConfirmLoading = true;
    if (!isDelete) {
      this.isDeleteLogVisible = false;
      this.deleteItems = [];
      return;
    }
    let toDelete = [];
    this.entries.forEach(e => {
      this.deleteItems.forEach(l => {
        if (e.batchId === l.batchId) {
          toDelete.push(
            this.http
              .delete(`/v1/logs/entries/${e.id}`, getHttpRequestOptions())
              .toPromise()
          );
        }
      });
    });
    Promise.all(toDelete).then(
      res => {
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteLogVisible = false;
        this.deleteItems = [];
        this.getLogs();
      },
      rej => {
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        handleAuthError(
          () => alert('Failed to delete log bundle'),
          rej,
          this.router,
          this.http,
          () => this.OnClickDeleteBundle(isDelete)
        );
      }
    );
  };

  uploadLogs = function(uploadLogs) {
    if (!uploadLogs || this.selectedEdges.length === 0) {
      this.isCreateLogVisible = false;
      return;
    }
    this.isUploading = true;
    this.uploaded = true;
    const edgeIds = [];
    this.selectedEdges.forEach(e => {
      edgeIds.push(e.id);
    });
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
          this.startFetchingLogs();
          this.isModalConfirmLoading = false;
          this.isCreateLogVisible = false;
          this.isUploading = false;
        },
        reject => {
          this.isModalConfirmLoading = false;
          this.isCreateLogVisible = false;
          this.isUploading = false;
          handleAuthError(
            () => alert('Failed to upload logs'),
            reject,
            this.router,
            this.http,
            () => this.uploadLogs(uploadLogs)
          );
        }
      );
  };

  startFetchingLogs() {
    this.getLogs();
    this.fetchLogsTimer = setInterval(() => this.getLogs(), 60000);
  }
  onSelectAllEdges = function() {
    if (this.selectAllEdges) {
      this.selectedEdges = this.connectedEdges.slice();
      this.connectedEdges.forEach(ele => {
        ele.selected = true;
        const edgeItem = this.allConnectedEdges.find(e => e.id === ele.id);
        if (edgeItem) {
          edgeItem.selected = true;
        }
      });
    } else {
      this.selectedEdges = [];
      this.connectedEdges.forEach(ele => {
        ele.selected = false;
        const edgeItem = this.allConnectedEdges.find(e => e.id === ele.id);
        if (edgeItem) {
          edgeItem.selected = false;
        }
      });
    }
  };

  onSelectEdge = function(edge) {
    const edgeItem = this.allConnectedEdges.find(e => e.id === edge.id);
    if (edge.selected) {
      this.selectedEdges.push(edge);
      if (edgeItem) {
        edgeItem.selected = true;
      }
    } else {
      if (edgeItem) {
        edgeItem.selected = false;
      }
      const index = this.selectedEdges.findIndex(ele => {
        ele.id = edge.id;
      });
      this.selectedEdges.splice(index, 1);
    }
  };

  showDeleteConfirm = function(logs) {
    this.deleteItems = logs.slice();
    this.isDeleteLogVisible = true;
  };

  onFilterChange = function() {
    const searchVal = this.searchVal.trim().toLowerCase();
    const newEdges = [];

    this.allEdges.forEach(e => {
      const name = e.name.toLowerCase();
      if (searchVal.length === 0 || name.indexOf(searchVal) > -1) {
        newEdges.push(e);
      }
    });

    if (this.isDownload) {
      for (let i = 0; i < this.relatedEdges.length; i++) {
        this.relatedEdges[i].hide = true;
        for (let j = 0; j < newEdges.length; j++) {
          if (this.relatedEdges[i].name === newEdges[j].name) {
            this.relatedEdges[i].hide = false;
            break;
          }
        }
      }
    } else {
      this.connectedEdges = [];
      this.selectedEdges = [];
      for (let i = 0; i < this.allConnectedEdges.length; i++) {
        for (let j = 0; j < newEdges.length; j++) {
          if (this.allConnectedEdges[i].id === newEdges[j].id) {
            this.connectedEdges.push(this.allConnectedEdges[i]);
            if (this.allConnectedEdges[i].selected) {
              this.selectedEdges.push(this.allConnectedEdges[i]);
            }
            break;
          }
        }
      }
    }
  };

  OnClickDeleteBundleCancel() {
    this.isConfirmLoading = false;
    this.isDeleteLogVisible = false;
    this.isDeleteLogVisible = false;
  }

  showStatus(log) {
    if (log.status === 'Failed') {
      return 'Details';
    }
    return 'Download';
  }
}
