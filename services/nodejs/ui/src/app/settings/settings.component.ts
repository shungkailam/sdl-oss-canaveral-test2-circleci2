import { Component, OnInit, OnDestroy, EventEmitter } from '@angular/core';
import { Router, ActivatedRoute, NavigationEnd } from '@angular/router';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import { TableBaseComponent } from '../base-components/table.base.component';
import { RegistryService } from '../services/registry.service';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.css'],
})
export class SettingsComponent extends TableBaseComponent
  implements OnInit, OnDestroy {
  settingsData = [];
  routerEventUrl = '/settings';
  whichprofile = 'Cloud';

  constructor(router: Router, private http: Http) {
    super(router);
    this.routerEventSubscription = this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        this.settingsData = [];
        if (event.url === '/settings/general') {
          try {
            this.whichprofile = 'Global Settings';
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, null);
          }
        }
        if (event.url === '/settings/container') {
          try {
            this.fetchProfilesData();
            this.whichprofile = 'Container Registry';
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, null);
          }
        }
        if (event.url === '/settings/clouds' || event.url === '/settings') {
          try {
            this.fetchData();
            this.whichprofile = 'Cloud';
          } catch (e) {
            handleAuthError(null, e, this.router, this.http, null);
          }
        }
      }
    });
  }

  ngOnInit() {}
  fetchData() {
    this.http
      .get('/v1/cloudcreds', getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          const data = x.json();
          this.settingsData = data;
        },
        err => {
          handleAuthError(null, err, this.router, this.http, () =>
            this.fetchData()
          );
        }
      );
  }
  fetchProfilesData() {
    this.http
      .get('/v1/containerregistries', getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          const data = x.json();
          this.settingsData = data;
        },
        err => {
          handleAuthError(null, err, this.router, this.http, () =>
            this.fetchProfilesData()
          );
        }
      );
  }

  ngOnDestroy() {
    this.unsubscribeRouterEventMaybe();
  }
}
