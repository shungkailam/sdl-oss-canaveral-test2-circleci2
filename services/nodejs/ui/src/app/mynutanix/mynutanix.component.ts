import { Component } from '@angular/core';
import {
  Router,
  ActivatedRoute,
  ParamMap,
  Params,
  NavigationEnd,
} from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../base-components/table.base.component';
import { AggregateInfo } from '../model/index';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../services/registry.service';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import * as jwt from 'jsonwebtoken';
@Component({
  templateUrl: './mynutanix.component.html',
  styleUrls: ['./mynutanix.component.css'],
  selector: 'app-mynutanix',
})
export class MyNutanixComponent extends TableBaseComponent {
  code = '';
  state = '';
  returnUrl = '';
  refreshToken = '';
  noSherlockAccess: boolean = false;

  constructor(
    router: Router,
    private regService: RegistryService,
    private route: ActivatedRoute,
    private http: Http
  ) {
    super(router);
  }

  ngOnInit() {
    this.route.queryParams.subscribe(params => {
      if (params['state'] && params['code']) {
        this.state = params['state'];
        this.code = params['code'];
        console.log('hit param object');
      }
      const cookies = document.cookie;
      let oauthState = '';
      let cookieEle = cookies.split(';');
      if (cookieEle && cookieEle.length > 0) {
        cookieEle.forEach(e => {
          const ele = e.split('=');
          if (ele) {
            const name = ele[0].trim();
            if (name === 'oauth-state') {
              oauthState = ele[1].trim();
            }
          }
        });
      }
      if (this.state && this.state === oauthState) {
        this.http
          .post(
            '/v1/oauth2/token',
            { code: this.code },
            getHttpRequestOptions()
          )
          .toPromise()
          .then(
            res => {
              const token = res.json().token;
              if (token) {
                localStorage['sherlock_auth_token'] = token;
                const decodedToken = jwt.decode(token);
                if (decodedToken) {
                  localStorage['sherlock_refresh_token'] =
                    decodedToken['refreshToken'];
                  localStorage['sherlock_mynutanix_email'] =
                    decodedToken['email'];
                  if (decodedToken['specialRole'] === 'admin') {
                    localStorage['sherlock_role'] = 'infra_admin';
                  } else {
                    localStorage['sherlock_role'] = '';
                  }
                  this.regService.register(
                    REG_KEY_TENANT_ID,
                    decodedToken['tenantId']
                  );
                }
              }
              if (this.state) {
                const decodedState = jwt.decode(this.state);
                if (decodedState) {
                  this.returnUrl = decodedState['returnUrl'];
                  if (this.returnUrl === 'undefined') {
                    this.returnUrl = 'edges';
                  }
                }
              }
              this.router.navigate(['/' + this.returnUrl]);
            },
            rej => {
              this.noSherlockAccess = true;
            }
          );
      } else {
        if (window.location.href.indexOf('login') === -1) {
          this.router.navigate(['/login'], {
            queryParams: { returnUrl: 'edges' },
          });
        }
        // go to error page
      }
    });
  }

  onClickLearnSherlock() {
    window.location.href = 'https://www.nutanix.com/products/iot/';
  }

  onClickmuNutanix() {
    window.location.href = 'https://my.nutanix.com/';
  }
}
