import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import * as uuidv4 from 'uuid/v4';
import { TableBaseComponent } from '../../../base-components/table.base.component';

@Component({
  selector: 'app-project-alerts',
  templateUrl: './project.alerts.component.html',
  styleUrls: ['./project.alerts.component.css'],
})
export class ProjectAlertsComponent {
  queryParamSub = null;
  projectId = '';
  routerEventUrl = '';
  constructor(
    private router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.id) {
        // id param exists - update case
        let project = this.regService.get(params.id);
        if (project) this.projectId = project.id;
        this.routerEventUrl = `/project/${this.projectId}/alerts`;
      }
    });
  }
}
