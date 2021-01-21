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
  selector: 'app-projects-summary',
  templateUrl: './summary.component.html',
  styleUrls: ['./summary.component.css'],
})
export class ProjectsSummaryComponent extends TableBaseComponent {
  columns = ['Name', 'Clouds', 'Edges', 'Storage Status', 'Compute Status'];
  data = [];
  isConfirmLoading = false;

  routerEventUrl = '/projects/summary';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService
  ) {
    super(router);
  }
}
