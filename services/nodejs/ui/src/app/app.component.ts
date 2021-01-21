import { Component } from '@angular/core';
import {
  Router,
  ActivatedRoute,
  ParamMap,
  NavigationEnd,
} from '@angular/router';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css'],
})
export class AppComponent {
  sub = null;
  constructor(private router: Router) {
    this.sub = this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        if (event.url == '/') {
          if (localStorage['sherlock_role'] !== '') router.navigate(['/edges']);
          else router.navigate(['/projects']);
        }
      }
    });
  }
}
