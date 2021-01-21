import { Component, OnInit, OnDestroy, EventEmitter } from '@angular/core';
import { Router, ActivatedRoute, NavigationEnd } from '@angular/router';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import { TableBaseComponent } from '../base-components/table.base.component';
import { RegistryService } from '../services/registry.service';

@Component({
  selector: 'app-projects',
  templateUrl: './projects.component.html',
  styleUrls: ['./projects.component.css'],
})
export class ProjectsComponent extends TableBaseComponent
  implements OnInit, OnDestroy {
  data = [];
  routerEventUrl = '/projects';

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
      .get('/v1/projects', getHttpRequestOptions())
      .toPromise()
      .then(x => {
        const data = x.json();
        this.data = data;
      });
  }

  ngOnDestroy() {
    this.unsubscribeRouterEventMaybe();
  }
}
