import { Component } from '@angular/core';
import { Router, ActivatedRoute, NavigationEnd } from '@angular/router';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';
import { TableBaseComponent } from '../../base-components/table.base.component';
import { RegistryService } from '../../services/registry.service';

@Component({
  selector: 'app-project',
  templateUrl: './project.component.html',
  styleUrls: ['./project.component.css'],
})
export class ProjectComponent extends TableBaseComponent {
  data = [];
  sub = null;
  projectId = '';
  projectName = '';
  project: any = {};
  whichEntity = '';
  projectData = [];
  associatedUser = false;
  routerEventSubscribe = null;

  constructor(
    router: Router,
    private http: Http,
    private route: ActivatedRoute,
    private registryService: RegistryService
  ) {
    super(router);
    this.fetchingData();
  }

  fetchingData() {
    this.sub = this.route.params.subscribe(async params => {
      this.projectId = params['id'];
      this.project = this.registryService.get(params['id']);

      if (this.project) this.projectName = this.project.name;
      else {
        this.fetchProjects();
      }
    });
    this.routerEventSubscribe = this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        this.projectData = [];
        if (event.url.includes(`/project/${this.projectId}/scripts`)) {
          try {
            this.whichEntity = 'Scripts';
            this.fetchScriptsData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/runtime`)) {
          try {
            this.whichEntity = 'Runtime Environments';
            this.fetchRuntimeData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/applications`)) {
          try {
            this.whichEntity = 'Applications';
            this.fetchAppsData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/datastreams`)) {
          try {
            this.whichEntity = 'Data Streams';
            this.fetchDatastreamsData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/datasources`)) {
          try {
            this.whichEntity = 'Data Sources';
            this.fetchDatasourcesData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/edges`)) {
          try {
            this.whichEntity = 'Edges';
            this.fetchEdgesData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/users`)) {
          try {
            this.whichEntity = 'Users';
            this.fetchUsersData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/summary`)) {
          try {
            this.whichEntity = 'Summary';
            this.fetchUsersData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
        if (event.url.includes(`/project/${this.projectId}/alerts`)) {
          try {
            this.whichEntity = 'Alerts';
            this.fetchAlertsData();
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchingData()
            );
          }
        }
      }
    });
  }
  async fetchScriptsData() {
    this.projectData = await this.http
      .get(`/v1/projects/${this.projectId}/scripts`, getHttpRequestOptions())
      .toPromise()
      .then(x => x.json());
  }
  async fetchAppsData() {
    this.projectData = await this.http
      .get(
        `/v1/projects/${this.projectId}/applications`,
        getHttpRequestOptions()
      )
      .toPromise()
      .then(x => x.json());
  }
  async fetchRuntimeData() {
    this.projectData = await this.http
      .get(
        `/v1/projects/${this.projectId}/scriptruntimes`,
        getHttpRequestOptions()
      )
      .toPromise()
      .then(x => x.json());
  }
  async fetchDatastreamsData() {
    this.projectData = await this.http
      .get(
        `/v1/projects/${this.projectId}/datastreams`,
        getHttpRequestOptions()
      )
      .toPromise()
      .then(x => x.json());
  }
  async fetchDatasourcesData() {
    this.projectData = await this.http
      .get(
        `/v1/projects/${this.projectId}/datasources`,
        getHttpRequestOptions()
      )
      .toPromise()
      .then(x => x.json());
  }
  async fetchEdgesData() {
    this.projectData = await this.http
      .get(`/v1/projects/${this.projectId}/edges`, getHttpRequestOptions())
      .toPromise()
      .then(x => x.json());
  }
  async fetchUsersData() {
    this.projectData = await this.http
      .get(`/v1/projects/${this.projectId}/users`, getHttpRequestOptions())
      .toPromise()
      .then(x => x.json());
  }
  async fetchProjects() {
    const projects = await this.http
      .get('/v1/projects', getHttpRequestOptions())
      .toPromise()
      .then(x => x.json());
    projects.some(p => {
      if (p.id === this.projectId) {
        this.project = p;
        this.projectName = p.name;
      }
    });
  }
  fetchAlertsData() {}
  ngOnDestroy() {
    this.sub.unsubscribe();
    this.routerEventSubscribe.unsubscribe();
    super.ngOnDestroy();
  }
}
