import { Component, OnDestroy, OnInit } from '@angular/core';

@Component({
  selector: 'app-edge-summary',
  templateUrl: './edge.summary.component.html',
  styleUrls: ['./edge.summary.component.css'],
})
export class EdgeSummaryComponent implements OnInit, OnDestroy {
  ngOnInit() {
    document.querySelector('body').classList.add('body-dashboard2');
  }

  ngOnDestroy() {
    document.querySelector('body').classList.remove('body-dashboard2');
  }
}
