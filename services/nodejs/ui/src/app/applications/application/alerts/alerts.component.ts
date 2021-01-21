import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { AggregateInfo } from '../../../model/index';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { Edge } from '../../../model/index';
import { reject } from 'q';
import { TableBaseComponent } from '../../../base-components/table.base.component';

@Component({
  selector: 'app-application-alerts',
  templateUrl: './alerts.component.html',
  styleUrls: ['./alerts.component.css'],
})
export class ApplicationAlertsComponent extends TableBaseComponent {
  isLoading = false;
  isDeleteModalVisible = false;
  toDelete = [];
  data = [];
  isConfirmLoading = false;

  columns = [
    'Name',
    'Edge',
    'Severity',
    'Resolved',
    'Acknowledged',
    'Current Time',
  ];
  routerEventUrl = '/applications/application/alerts';

  sortMap = {
    Name: null,
    Edge: null,
    Severity: null,
    Resolved: null,
    Acknowledged: null,
    'Current Time': null,
  };

  mapping = {
    Name: 'name',
    Edge: 'edge',
    Severity: 'severity',
    Resolved: 'resolved',
    Acknowledged: 'ack',
    'Current Time': 'time',
  };

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
  }
  fetchData() {}
  onClickackAlerts() {}
  onClickResoleAlerts() {}
  onClickdownloadLogs() {}
}
