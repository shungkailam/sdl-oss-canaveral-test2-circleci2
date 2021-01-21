import { Injectable } from '@angular/core';
import { Http } from '@angular/http';
import { AuthService } from '../guards/auth.service';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../services/registry.service';
import * as jwt from 'jsonwebtoken';

@Injectable()
export class LoginService {
  constructor(
    private http: Http,
    private regService: RegistryService,
    private authService: AuthService
  ) {}

  login({ email, password }) {
    return this.http
      .post('/v1/login', { email, password })
      .toPromise()
      .then((x: any) => {
        // this.updateTenantIdInRegistry();
        this.authService.setUser(email);
        // also save the creds in localStorage so no need to login again
        localStorage['sherlock_creds'] = JSON.stringify({
          password,
          username: email,
        });
        const { token } = x.json();
        localStorage['sherlock_auth_token'] = token;
        const decoded: any = jwt.decode(token);
        console.log('login token decoded:', decoded);
        if (decoded && decoded['specialRole'] === 'admin') {
          localStorage['sherlock_role'] = 'infra_admin';
        } else {
          localStorage['sherlock_role'] = '';
        }
        this.regService.register(REG_KEY_TENANT_ID, decoded.tenantId);
        return x;
      });
  }
}
