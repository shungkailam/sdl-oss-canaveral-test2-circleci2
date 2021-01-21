import { Component } from '@angular/core';
import { Router, ActivatedRoute, NavigationEnd } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../base-components/table.base.component';
import { RegistryService } from '../../services/registry.service';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';
import * as uuidv4 from 'uuid/v4';

@Component({
  selector: 'app-ota-inprogress',
  templateUrl: './ota.inProgress.component.html',
  styleUrls: ['./ota.inProgress.component.css'],
})
export class OtaInProgressComponent extends TableBaseComponent {
  progress = [];
  isLoading = false;
  edgeInfo = [];
  edges = [];
  showProgress = false;
  updateTitle = '';
  selectedEntitiesText = '';
  extraText = '';
  fetchProgress = null;
  edgeUpdate = [];

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.getInProgress();
    this.fetchProgress = setInterval(() => {
      this.getInProgress();
    }, 10000);
  }

  getInProgress() {
    const body = {
      path: '/edge.*/upgrade.*',
    };
    this.isLoading = true;
    console.log('11');
    let promise = [];
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.post('/v1/events', body, getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 2 && res[1].json().length > 0) {
          const edge = res[0].json();
          this.edgeUpdate = res[1].json();
          let percentageSum = 0;
          let successEdgeNum = 0;
          this.progress[0] = {
            title: 'Edge Update',
            version: '',
            edges: [],
            percentage: '',
            status: 'Upgrading',
          };
          this.edgeUpdate.forEach(p => {
            if (p.path) {
              let path = p.path;
              let item = path.split(':');
              let edgeEntity = {
                name: '',
                progress: p.message,
                status: p.state,
              };
              percentageSum += parseFloat(p.message);
              if (item.length === 5) {
                const edgeInfo = item[1];
                this.progress[0]['version'] = item[2];
                const edgeItem = edgeInfo.split('/');
                if (edgeItem.length === 2) {
                  let edgeId = edgeItem[0];
                  const edge1 = edge.find(e => e.id === edgeId);
                  if (edge1) {
                    edgeEntity['name'] = edge1.name;
                  }
                }
              }
              this.progress[0]['edges'].push(edgeEntity);
              if (p.state === 'Failed') {
                this.progress[0].status = 'Failed';
              } else {
                successEdgeNum++;
              }

              if (p.state === 'Acknowledged') {
                this.progress[0].status = 'Acknowledged';
              }
            }
          });
          if (successEdgeNum > 0) {
            this.progress[0].percentage =
              Math.round(percentageSum / successEdgeNum) + '%';
          } else {
            this.progress[0].percentage = '0%';
          }

          if (
            this.progress[0].percentage === '100%' &&
            this.progress[0].status !== 'Acknowledged'
          ) {
            this.progress[0].status = 'Complete';
          }
          this.setEntitiesText();
        } else {
          clearInterval(this.fetchProgress);
        }

        this.isLoading = false;
      },

      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.getInProgress()
        );
        this.isLoading = false;
        clearInterval(this.fetchProgress);
      }
    );
  }

  getWidth(percentage) {
    return `${parseFloat(percentage)}%`;
  }

  setEntitiesText() {
    if (this.progress[0].edges.length > 0) {
      this.selectedEntitiesText = 'Edge ' + this.progress[0].edges[0].name;
    } else {
      this.selectedEntitiesText = 'No Entities';
    }

    if (this.progress[0].edges.length > 1) {
      this.extraText = ' and ' + (this.progress[0].edges.length - 1) + ' more';
    } else {
      this.extraText = '';
    }
  }

  viewEntities() {
    this.showProgress = true;
    this.updateTitle =
      'Selected edges for Edge Update - ' +
      this.progress[0].version +
      ' update.';
  }

  OnHideProgress() {
    this.showProgress = false;
  }

  acknowledgeUpdate() {
    this.edgeUpdate.forEach(e => {
      e.state = 'Acknowledged';
    });
    const body = {
      events: this.edgeUpdate,
    };
    this.http
      .put('/v1/events', body, getHttpRequestOptions())
      .toPromise()
      .then(
        res => {
          this.getInProgress();
        },

        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.acknowledgeUpdate()
          );
        }
      );
  }

  ngOnDestroy() {
    clearInterval(this.fetchProgress);
    super.ngOnDestroy();
  }
}
