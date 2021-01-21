import { Component, OnDestroy, OnInit } from '@angular/core';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
})
export class DashboardComponent implements OnInit, OnDestroy {
  ngOnInit() {
    document.querySelector('body').classList.add('body-dashboard');
  }

  ngOnDestroy() {
    document.querySelector('body').classList.remove('body-dashboard');
  }
}
