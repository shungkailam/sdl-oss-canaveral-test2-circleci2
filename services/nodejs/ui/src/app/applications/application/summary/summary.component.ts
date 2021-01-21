import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { AggregateInfo } from '../../../model/index';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { Edge } from '../../../model/index';
import { reject } from 'q';

@Component({
  selector: 'app-applications-summary',
  templateUrl: './summary.component.html',
  styleUrls: ['./summary.component.css'],
})
export class ApplicationSummaryComponent extends TableBaseComponent {
  sub = null;
  data = [];
  appId = '';
  app: any = null;
  appName = '';
  appDesc = '';
  appCont = [];
  appEdges = 0;
  edges = [];
  isLoading = false;

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private registryService: RegistryService,
    private http: Http
  ) {
    super(router);
    this.sub = this.route.parent.params.subscribe(params => {
      this.appId = params['id'];
      this.routerEventUrl = `/application/${this.appId}/summary`;
      this.app = this.registryService.get(params['id']);
      if (this.app) {
        this.appName = this.app.name;
        this.appDesc = this.app.description;
      } else {
        this.appId = params['id'];
        this.http
          .get(`/v1/application/${this.appId}`, getHttpRequestOptions())
          .toPromise()
          .then(
            res => {
              const app = res.json();
              this.appName = app.name;
              this.appDesc = app.description;
            },
            rej => {
              handleAuthError(null, rej, this.router, this.http, null);
            }
          );
      }
    });
  }
  async fetchData() {
    this.isLoading = true;
    let promise = [];
    let path = '/edge:.*';
    let event = {
      id: '',
      message: '',
      path: path,
      source_type: '',
      state: '',
      timestamp: '',
      type: '',
      version: '',
    };
    promise.push(
      this.http.get('/v1/applications', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http
        .get('/v1/applicationstatus', getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.post('/v1/events', event, getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 4) {
          const appData = res[0].json();
          const statusData = res[1].json();
          const edge = res[2].json();
          const events = res[3].json();
          appData.some(app => {
            if (app.id === this.appId) {
              this.app = app;
              this.appName = app.name;
              this.appDesc = app.description;
            }
          });
          console.log(events);
          const appEdges = [];
          edge.forEach(e => {
            if (this.app && this.app.edgeIds) {
              const appEntry = this.app.edgeIds.find(ae => ae === e.id);
              if (appEntry) {
                appEdges.push(e);
              }
            }
          });
          const edgesData = appEdges;
          this.edges = appEdges;
          let podName = '';
          statusData.forEach(s => {
            if (s.applicationId === this.appId) {
              this.appCont = [];
              edgesData.forEach(e => {
                if (e.id === s.edgeId) {
                  let cRunning = 0;
                  let cTotal = 0;
                  if (s.appStatus.podStatusList !== null) {
                    s.appStatus.podStatusList.forEach(p => {
                      if (
                        p.status.containerStatuses &&
                        p.status.containerStatuses !== null
                      ) {
                        p.status.containerStatuses.forEach(c => {
                          cTotal++;
                          if (
                            podName.trim().toLowerCase() !==
                            p.metadata.name.trim().toLowerCase()
                          ) {
                            this.appCont.push({ name: c.name });
                          }

                          if (c.state.running && c.state.running !== null) {
                            cRunning++;
                          }
                        });
                      }
                    });
                  }

                  e['status'] = cRunning + ' of ' + cTotal + ' Running';
                  e['alerts'] = '-';
                }
              });
            }
            this.appEdges = edgesData.length;
            this.isLoading = false;
          });
        } else {
          this.isLoading = false;
        }
      },

      rej => {
        this.isLoading = false;
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchData()
        );
      }
    );
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
    super.ngOnDestroy();
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
  onClickDeployDetails() {
    this.registryService.register(this.appId, this.app);
    this.router.navigate(['/application', this.appId, 'deployment'], {
      relativeTo: this.route,
    });
  }
}
