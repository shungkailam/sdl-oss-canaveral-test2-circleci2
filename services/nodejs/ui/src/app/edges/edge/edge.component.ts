import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import { RegistryService } from '../../services/registry.service';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';

@Component({
  selector: 'app-edge',
  templateUrl: './edge.component.html',
  styleUrls: ['./edge.component.css'],
})
export class EdgeComponent implements OnInit, OnDestroy {
  edgeId: string = null;
  sub = null;
  edge: any = {};

  datasources = [];

  constructor(
    private router: Router,
    private http: Http,
    private route: ActivatedRoute,
    private registryService: RegistryService
  ) {}

  ngOnInit() {
    this.sub = this.route.params.subscribe(async params => {
      this.edgeId = params['id'];
      let promise = [];
      promise.push(
        this.http
          .get(`/v1/edges/${this.edgeId}`, getHttpRequestOptions())
          .toPromise()
      );
      promise.push(
        this.http
          .get(`/v1/edges/${this.edgeId}/datasources`, getHttpRequestOptions())
          .toPromise()
      );
      Promise.all(promise).then(
        response => {
          this.edge = response[0].json();
          this.datasources = response[1].json();
        },
        reject => {
          handleAuthError(null, reject, this.router, this.http, () =>
            this.ngOnInit()
          );
        }
      );
    });
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }
}
