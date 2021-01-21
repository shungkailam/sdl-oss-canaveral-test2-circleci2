import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';

@Component({
  selector: 'app-datasources',
  templateUrl: './datasources.component.html',
  styleUrls: ['./datasources.component.css'],
})
export class DataSourcesComponent {
  data = [];

  constructor(private router: Router, private http: Http) {
    this.fetchData();
  }

  fetchData() {
    this.http
      .get('/v1/datasources', getHttpRequestOptions())
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
}
