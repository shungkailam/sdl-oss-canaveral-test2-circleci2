import { Injectable } from '@angular/core';

// skip welcome for .Next demo
const SKIP_WELCOME = true;

@Injectable()
export class OnBoardService {
  private getKey(email: string): string {
    return `onboarded:${email}`;
  }
  public isOnBoarded(email: string): boolean {
    if (SKIP_WELCOME) {
      return true;
    } else {
      const key = this.getKey(email);
      return localStorage.getItem(key) === 'true';
    }
  }
  public setOnBoarded(email: string, b: boolean) {
    if (SKIP_WELCOME) {
      return;
    }
    const key = this.getKey(email);
    if (b) {
      localStorage.setItem(key, 'true');
    } else {
      localStorage.removeItem(key);
    }
  }
}
