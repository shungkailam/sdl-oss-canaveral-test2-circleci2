import { Headers, RequestOptions } from '@angular/http';

export function getHttpRequestOptions(): RequestOptions {
  const token = localStorage['sherlock_auth_token'];
  if (token) {
    const headers = new Headers();
    headers.append('authorization', `Bearer ${token}`);
    return new RequestOptions({ headers: headers });
  } else {
    return new RequestOptions();
  }
}
