import { Injectable } from '@angular/core';
import {
  Router,
  CanActivate,
  ActivatedRouteSnapshot,
  RouterStateSnapshot,
} from '@angular/router';
import { Http } from '@angular/http';
import { AuthService } from './auth.service';
import { LoginService } from '../login/login.service';

@Injectable()
export class AuthGuard implements CanActivate {
  loginPromise = null;

  constructor(
    private router: Router,
    private authService: AuthService,
    private loginService: LoginService,
    private http: Http
  ) {
    const creds = localStorage['sherlock_creds'];
    if (creds) {
      try {
        const credsObj = JSON.parse(creds);
        if (credsObj.username && credsObj.password) {
          this.loginPromise = this.loginService.login({
            email: credsObj.username,
            password: credsObj.password,
          });
        }
      } catch (e) {
        // ignore
      }
    } else if (localStorage['sherlock_mynutanix_email']) {
      authService.setUser(localStorage['sherlock_mynutanix_email']);
    }
  }

  canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot) {
    if (this.loginPromise) {
      return this.loginPromise.then(
        () => {
          this.loginPromise = null;
          return this.canActivate(route, state);
        },
        err => {
          this.loginPromise = null;
          return this.canActivate(route, state);
        }
      );
    }
    if (this.authService.isAuthenticated()) {
      return true;
    }

    // not logged in so redirect to login page with the return url
    this.router.navigate(['/login'], { queryParams: { returnUrl: state.url } });
    return false;
  }
}
