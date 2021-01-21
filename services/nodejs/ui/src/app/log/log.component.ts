import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../base-components/table.base.component';
import { AggregateInfo } from '../model/index';
import { RegistryService } from '../services/registry.service';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import { Edge } from '../model/index';

@Component({
  selector: 'app-log',
  templateUrl: './log.component.html',
  styleUrls: ['./log.component.css'],
})
export class LogComponent extends TableBaseComponent {
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
    Time: 'date',
    'Collected for': 'count',
    Status: 'status',
  };

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
  }

  ngOnDestroy() {
    clearInterval(this.fetchLogsTimer);
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
          const entries = res[0].json();
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
            this.isLoading = false;
          }
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
    this.connectedEdges = [];
    this.http
      .get('/v1/edges', getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          this.edges = response.json();
          this.edges.forEach(ele => {
            ele.selected = false;
            if (ele.connected) this.connectedEdges.push(ele);
          });
          this.allEdges = this.edges.slice();
          this.selectedEdges = [];
          this.allConnectedEdges = this.connectedEdges.slice();
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

  async fetchData() {
    this.getLogs();
    this.fetchEdges();
  }

  onClickCreateLog = function() {
    this.isPopupLoading = true;
    this.isCreateLogVisible = true;
    this.fetchEdges().then(v => {
      this.selectAllEdges = false;
      this.selectedEdges = [];
      this.isDownload = false;
      this.searchVal = '';
      this.uploaded = false;
      this.edges = this.allEdges;
      this.edges.forEach(e => {
        e.selected = false;
      });
      this.isPopupLoading = false;
    });
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
              this.isCreateLogVisible = true;
              this.isDownload = true;
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
                let r = res.json();
                this.isPopupLoading = false;
                this.isCreateLogVisible = true;
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
    this.isConfirmLoading = true;
    this.isDeleteLogVisible = true;
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
        this.getLogs();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.deleteItems = [];
        this.isDeleteLogVisible = false;
      },
      rej => {
        handleAuthError(
          () => alert('Failed to delete logs'),
          rej,
          this.router,
          this.http,
          () => this.OnClickDeleteBundle(isDelete)
        );
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteLogVisible = false;
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
        },
        getHttpRequestOptions()
      )
      .toPromise()
      .then(
        response => {
          this.startFetchingLogs();
          this.isCreateLogVisible = false;
          this.isUploading = false;
        },
        reject => {
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
      edgeItem.selected = true;
    } else {
      edgeItem.selected = false;
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
    this.isModalConfirmLoading = false;
    this.isDeleteLogVisible = false;
  }

  showStatus(log) {
    if (log.status === 'Failed') {
      return 'Details';
    }
    return 'Download';
  }
}
