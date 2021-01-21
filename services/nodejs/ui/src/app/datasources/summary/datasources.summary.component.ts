import { Component, OnDestroy, OnInit } from '@angular/core';

@Component({
  selector: 'app-datasources-summary',
  templateUrl: './datasources.summary.component.html',
  styleUrls: ['./datasources.summary.component.css'],
})
export class DataSourcesSummaryComponent implements OnInit, OnDestroy {
  ngOnInit() {
    document.querySelector('body').classList.add('body-dashboard2');
  }

  ngOnDestroy() {
    document.querySelector('body').classList.remove('body-dashboard2');
  }
}
