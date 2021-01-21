import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, NavigationEnd } from '@angular/router';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import { TableBaseComponent } from '../base-components/table.base.component';

@Component({
  selector: 'app-datastreams',
  templateUrl: './datastreams.component.html',
  styleUrls: ['./datastreams.component.css'],
})
export class DataStreamsComponent extends TableBaseComponent
  implements OnInit, OnDestroy {
  data = [];
  routerEventUrl = '/datastreams';
  constructor(router: Router, private http: Http) {
    super(router);
    this.routerEventSubscription = this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        try {
          this.fetchData();
        } catch (e) {
          handleAuthError(null, e, this.router, this.http, null);
        }
      }
    });
  }

  ngOnInit() {}
  fetchData() {
    this.http
      .get('/v1/datastreams', getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          const data = x.json();
          this.data = data;
        },
        e => {
          handleAuthError(null, e, this.router, this.http, () =>
            this.fetchData()
          );
        }
      );
  }
  ngOnDestroy() {
    this.unsubscribeRouterEventMaybe();
  }
}
