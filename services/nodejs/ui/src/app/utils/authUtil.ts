import { Router } from '@angular/router';
import { Http } from '@angular/http';
import * as jwt from 'jsonwebtoken';

export function handleAuthError(
  customCallback,
  e,
  router: Router,
  http: Http,
  callback
): boolean {
  if (e.status === 401) {
    if (!localStorage['sherlock_refresh_token']) {
      if (localStorage['sherlock_creds']) {
        const userObj = JSON.parse(localStorage['sherlock_creds']);
        const email = userObj.username;
        const password = userObj.password;
        http
          .post('/v1/login', { email, password })
          .toPromise()
          .then(
            res => {
              localStorage['sherlock_auth_token'] = res.json().token;
              if (callback && typeof callback === 'function') {
                callback();
              }
            },
            rej => {
              console.log(rej);
              if (window.location.href.indexOf('login') === -1) {
                router.navigate(['/login'], {
                  queryParams: { returnUrl: router.url },
                });
              }
            }
          );
      } else {
        if (window.location.href.indexOf('login') === -1) {
          router.navigate(['/login'], {
            queryParams: { returnUrl: router.url },
          });
        }
      }
    } else {
      const newToken = localStorage['sherlock_refresh_token'];
      http
        .post('/v1/oauth2/token', { refreshToken: newToken })
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
              }

              if (callback && typeof callback === 'function') {
                callback();
              }
            } else {
              if (window.location.href.indexOf('login') === -1) {
                router.navigate(['/login'], {
                  queryParams: { returnUrl: router.url },
                });
              }
            }
          },

          rej => {
            console.log('error');
            if (window.location.href.indexOf('login') === -1) {
              router.navigate(['/login'], {
                queryParams: { returnUrl: router.url },
              });
            }
          }
        );
    }
    return true;
  } else {
    if (customCallback && typeof customCallback === 'function') {
      customCallback();
    }
    console.warn('ignore non-auth error:', e);
    return false;
  }
}
