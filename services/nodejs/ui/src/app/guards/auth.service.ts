import { Injectable } from '@angular/core';

@Injectable()
export class AuthService {
  user = '';

  public isAuthenticated(): boolean {
    // hack to skip auth when serving only UI via 'ng serve'
    if (window.location.href.match(/4200/)) {
      return true;
    }

    if (localStorage['sherlock_refresh_token']) {
      return true;
    }

    return this.user !== '';
  }

  public setUser(user: string) {
    this.user = user || '';
  }

  public getUser(): string {
    return this.user;
  }
}
