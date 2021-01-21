import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../base-components/table.base.component';
import { RegistryService } from '../../services/registry.service';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import * as uuidv4 from 'uuid/v4';

@Component({
  selector: 'app-settings',
  templateUrl: './general.component.html',
  styleUrls: ['./general.component.css'],
})
export class GeneralComponent {}
