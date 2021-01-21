import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { TableBaseComponent } from '../../../base-components/table.base.component';

@Component({
  selector: 'app-ota-confirm-update-popup',
  templateUrl: './ota.confirm-update.popup.component.html',
  styleUrls: ['./ota.confirm-update.popup.component.css'],
})
export class OtaConfirmUpdatePopupComponent extends TableBaseComponent {
  queryParamSub = null;
  updateInfo = [];
  updates = [];
  updateVersion = '';
  edges = [];
  selectedEntitiesText = 'No Entities';
  extraText = '';
  showEntities = false;
  updateTitle = '';

  constructor(
    private route: ActivatedRoute,
    router: Router,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.id) {
        this.updateInfo = regService.get(params.id);
        console.log(this.updateInfo);
        if (this.updateInfo) {
          this.updates = this.updateInfo['updates'];
          this.updateVersion = this.updateInfo['updateVersion'];
          this.edges = this.updateInfo['edge'];
          if (this.edges && this.edges.length > 0) {
            this.selectedEntitiesText = 'Edge ' + this.edges[0].name;
          } else {
            this.selectedEntitiesText = 'No Entities';
          }

          if (this.edges.length > 1) {
            this.extraText = ' and ' + (this.edges.length - 1) + ' more';
          } else {
            this.extraText = '';
          }
        }
      }
    });
  }

  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onConfirmUpdate() {
    console.log('update is confirmed');
    const body = {
      release: this.updateVersion,
      edgeIds: [],
    };
    this.edges.forEach(e => {
      body.edgeIds.push(e.id);
    });
    this.http
      .post('/v1/edges/upgrade', body, getHttpRequestOptions())
      .toPromise()
      .then(
        res => {
          this.router.navigate([{ outlets: { popup: null } }]);
        },

        rej => {
          this.router.navigate([{ outlets: { popup: null } }]);
          handleAuthError(
            alert(rej.json().message),
            rej,
            this.router,
            this.http,
            () => this.onConfirmUpdate()
          );
        }
      );
  }

  openEntities() {
    this.showEntities = true;
    this.updateTitle =
      'Select edges for Edge Update - ' + this.updateVersion + ' update.';
  }

  viewEntities() {
    this.showEntities = false;
  }
}
