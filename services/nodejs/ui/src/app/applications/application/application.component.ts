import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute, NavigationEnd } from '@angular/router';
import { Http } from '@angular/http';
import { RegistryService } from '../../services/registry.service';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';
import { TableBaseComponent } from '../../base-components/table.base.component';

@Component({
  selector: 'app-application',
  templateUrl: './application.component.html',
  styleUrls: ['./application.component.css'],
})
export class ApplicationComponent extends TableBaseComponent
  implements OnInit, OnDestroy {
  isLoading = false;
  isDeleteModalVisible = false;
  toDelete = [];
  data = [];
  columns = ['Name', 'Description', 'Last Updated'];
  routerEventUrl = '/applications/application';
  sub = null;
  appId = '';
  appName = '';
  app: any = null;
  edges = [];
  edgeCount = 0;
  logData = [];
  logsEntity = false;
  routerEventSubscribe = null;

  constructor(
    router: Router,
    private http: Http,
    private route: ActivatedRoute,
    private registryService: RegistryService
  ) {
    super(router);
    this.sub = this.route.params.subscribe(params => {
      this.appId = params.id;
      this.routerEventUrl = `/application/${this.appId}/summary`;
      this.app = this.registryService.get(params['id']);
      if (this.app) {
        this.appId = this.app.id;
        this.appName = this.app.name;
      }
    });
    this.routerEventSubscribe = this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        if (
          this.appId &&
          (event.url === `/application/${this.appId}/summary` ||
            `/application/${this.appId}/deployment`)
        ) {
          try {
            this.logsEntity = false;
            this.fetchData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchData()
            );
          }
        }
        if (this.appId && event.url === `/application/${this.appId}/logs`) {
          try {
            this.logsEntity = true;
            this.fetchLogsData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchLogsData()
            );
          }
        }
      }
    });
  }
  ngOnInit() {}
  async fetchData() {
    this.isLoading = true;
    try {
      const data = await this.http
        .get('/v1/applicationstatus', getHttpRequestOptions())
        .toPromise()
        .then(x => x.json());

      const edges = await this.http
        .get('/v1/edges', getHttpRequestOptions())
        .toPromise()
        .then(x => x.json());

      this.data = data;
      this.edgeCount = 0;
      this.data.forEach(s => {
        if (s.applicationId === this.appId) {
          edges.forEach(e => {
            if (e.id === s.edgeId) {
              this.edgeCount++;
            }
          });
        }
      });
      this.isLoading = false;
    } catch (e) {
      this.isLoading = false;
      handleAuthError(null, e, this.router, this.http, () => this.fetchData());
    }
  }
  async fetchLogsData() {
    this.logData = await this.http
      .get('/v1/logs/entries', getHttpRequestOptions())
      .toPromise()
      .then(x =>
        x.json().filter(e => e.tags[0] && e.tags[0].value === this.appId)
      );
  }
  ngOnDestroy() {
    this.sub.unsubscribe();
    this.routerEventSubscribe.unsubscribe();
    super.ngOnDestroy();
  }
}
