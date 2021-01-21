import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../services/registry.service';
import { TableBaseComponent } from '../../base-components/table.base.component';
import { getHttpRequestOptions } from '../../utils/httpUtil';

@Component({
  selector: 'app-projects-alerts',
  templateUrl: './alerts.component.html',
  styleUrls: ['./alerts.component.css'],
})
export class ProjectsAlertsComponent extends TableBaseComponent {
  columns = ['Name', 'Clouds', 'Edges', 'Storage Status', 'Compute Status'];
  data = [];
  isConfirmLoading = false;

  routerEventUrl = '/projects/alerts';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService
  ) {
    super(router);
  }
}
