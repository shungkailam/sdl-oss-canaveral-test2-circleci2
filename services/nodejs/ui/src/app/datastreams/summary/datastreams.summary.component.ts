import { Component, OnDestroy, OnInit } from '@angular/core';

@Component({
  selector: 'app-datastreams-summary',
  templateUrl: './datastreams.summary.component.html',
  styleUrls: ['./datastreams.summary.component.css'],
})
export class DataStreamsSummaryComponent implements OnInit, OnDestroy {
  ngOnInit() {
    document.querySelector('body').classList.add('body-dashboard2');
  }

  ngOnDestroy() {
    document.querySelector('body').classList.remove('body-dashboard2');
  }
}
