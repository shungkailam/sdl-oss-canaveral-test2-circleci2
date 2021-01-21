import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { Edge } from '../../../model/edge';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { TableBaseComponent } from '../../../base-components/table.base.component';

@Component({
  selector: 'app-edges-create-edge-popup',
  templateUrl: './edges.create-edge.popup.component.html',
  styleUrls: ['./edges.create-edge.popup.component.css'],
})
export class EdgesCreateEdgePopupComponent extends TableBaseComponent {
  queryParamSub = null;
  edge = null;
  edgeId = null;
  isConfirmLoading = false;
  duplicateEdgeFound = false;
  edgeName = '';
  edgeSerialNumber = '';
  edgeIPAddress = '';
  edgeSubnetMask = '';
  edgeGateway = '';
  invalidInput = false;
  edgeIP1 = { num: '', error: false };
  edgeIP2 = { num: '', error: false };
  edgeIP3 = { num: '', error: false };
  edgeIP4 = { num: '', error: false };
  subnetIP1 = { num: '', error: false };
  subnetIP2 = { num: '', error: false };
  subnetIP3 = { num: '', error: false };
  subnetIP4 = { num: '', error: false };
  gateIP1 = { num: '', error: false };
  gateIP2 = { num: '', error: false };
  gateIP3 = { num: '', error: false };
  gateIP4 = { num: '', error: false };
  edges = [];
  invalidEdgeName = false;
  isLoading = false;
  context = '';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.isLoading = true;
    this.fetchEdges();
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.projectId) {
        this.context = 'project';
      }
      if (params && params.id) {
        // id param exists - update case
        const edge: Edge = this.regService.get(params.id);
        if (edge) {
          this.edge = edge;
          this.edgeId = params.id;
          this.edgeName = edge.name;
          this.edgeSerialNumber = edge.serialNumber;
          this.edgeIPAddress = edge.ipAddress;
          this.edgeSubnetMask = edge.subnet;
          this.edgeGateway = edge.gateway;
          let ipArr = this.edgeIPAddress.split('.');
          let subnetArr = this.edgeSubnetMask.split('.');
          let gateArr = this.edgeGateway.split('.');
          this.edgeIP1.num = ipArr[0];
          this.edgeIP2.num = ipArr[1];
          this.edgeIP3.num = ipArr[2];
          this.edgeIP4.num = ipArr[3];
          this.subnetIP1.num = subnetArr[0];
          this.subnetIP2.num = subnetArr[1];
          this.subnetIP3.num = subnetArr[2];
          this.subnetIP4.num = subnetArr[3];
          this.gateIP1.num = gateArr[0];
          this.gateIP2.num = gateArr[1];
          this.gateIP3.num = gateArr[2];
          this.gateIP4.num = gateArr[3];
        }
      }
    });
  }
  async fetchEdges() {
    await this.http
      .get('/v1/edges', getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          this.edges = response.json();
          this.isLoading = false;
        },
        error => {
          handleAuthError(null, error, this.router, this.http, () =>
            this.fetchEdges()
          );
          this.isLoading = false;
        }
      );
  }
  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onCreateEdge() {
    this.isConfirmLoading = true;
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    const id = uuidv4();
    const edge = {
      id,
      tenantId,
      name: this.edgeName,
      serialNumber: this.edgeSerialNumber,
      ipAddress: this.edgeIPAddress,
      gateway: this.edgeGateway,
      subnet: this.edgeSubnetMask,
      edgeDevices: 0,
      storageCapacity: 0,
      storageUsage: 0,
      connected: false,
      NumCPU: '-',
      TotalMemory: '-',
      TotalStorage: '-',
      GPUInfo: '-',
      CPUUsage: '-',
      MemoryFree: '-',
      StorageFree: '-',
      GPUUsage: '-',
    };
    let method = 'post';
    if (this.edge !== null) {
      edge['id'] = this.edge.id;
      method = 'put';
    }
    this.http[method]('/v1/edges', edge, getHttpRequestOptions())
      .toPromise()
      .then(
        r => {
          this.isConfirmLoading = false;
          this.router.navigate([{ outlets: { popup: null } }]);
        },
        err => {
          this.isConfirmLoading = false;
          const warning =
            'Failed to ' + (method === 'post' ? 'create' : 'update') + ' edges';
          handleAuthError(
            () => alert(warning),
            err,
            this.router,
            this.http,
            () => this.onCreateEdge()
          );
          this.isConfirmLoading = false;
        }
      );
  }

  isCreateDisabled(): boolean {
    return (
      !this.edgeIP1.num ||
      !this.edgeIP2.num ||
      !this.edgeIP3.num ||
      !this.edgeIP4.num ||
      !this.subnetIP1.num ||
      !this.subnetIP2.num ||
      !this.subnetIP3.num ||
      !this.subnetIP4.num ||
      !this.gateIP1.num ||
      !this.gateIP2.num ||
      !this.gateIP3.num ||
      !this.gateIP4.num ||
      this.edgeIP1.error ||
      this.edgeIP2.error ||
      this.edgeIP3.error ||
      this.edgeIP4.error ||
      this.subnetIP1.error ||
      this.subnetIP2.error ||
      this.subnetIP3.error ||
      this.subnetIP4.error ||
      this.gateIP1.error ||
      this.gateIP2.error ||
      this.gateIP3.error ||
      this.gateIP4.error ||
      this.duplicateEdgeFound ||
      !this.edgeName ||
      !this.edgeSerialNumber ||
      !this.edgeIPAddress ||
      !this.edgeSubnetMask ||
      !this.edgeGateway
    );
  }
  validateIP(entity, idx) {
    let regex = /^(\d|[1-9]\d|1\d\d|2([0-4]\d|5[0-5]))$/;
    if (entity.num.length !== 0 && entity.num.includes('.')) {
      let entities = entity.num.split('.');
      if (entities.length === 4) {
        entities.forEach(e => {
          if (e.match(regex)) {
          }
        });
        if (idx === 'ip') {
          this.edgeIPAddress = '';
          this.edgeIP1.num = entities[0];
          this.edgeIP2.num = entities[1];
          this.edgeIP3.num = entities[2];
          this.edgeIP4.num = entities[3];
          entities.forEach(e => {
            this.edgeIPAddress += e + '.';
          });

          this.edgeIPAddress = this.edgeIPAddress.substr(
            0,
            this.edgeIPAddress.length - 1
          );
        }
        if (idx === 'sub') {
          this.edgeSubnetMask = '';
          this.subnetIP1.num = entities[0];
          this.subnetIP2.num = entities[1];
          this.subnetIP3.num = entities[2];
          this.subnetIP4.num = entities[3];
          entities.forEach(e => {
            this.edgeSubnetMask += e + '.';
          });

          this.edgeSubnetMask = this.edgeSubnetMask.substr(
            0,
            this.edgeSubnetMask.length - 1
          );
        }
        if (idx === 'gate') {
          this.edgeGateway = '';
          this.gateIP1.num = entities[0];
          this.gateIP2.num = entities[1];
          this.gateIP3.num = entities[2];
          this.gateIP4.num = entities[3];
          entities.forEach(e => {
            this.edgeGateway += e + '.';
          });

          this.edgeGateway = this.edgeGateway.substr(
            0,
            this.edgeGateway.length - 1
          );
        }
      } else {
        entity.error = true;
        this.invalidInput = true;
      }
      return;
    }
    if (entity.num.length !== 0 && !entity.num.match(regex)) {
      entity.error = true;
      this.invalidInput = true;
      return;
    }
    if (idx === 'ip') {
      this.edgeIPAddress =
        this.edgeIP1.num +
        '.' +
        this.edgeIP2.num +
        '.' +
        this.edgeIP3.num +
        '.' +
        this.edgeIP4.num;
    }
    if (idx === 'sub') {
      this.edgeSubnetMask =
        this.subnetIP1.num +
        '.' +
        this.subnetIP2.num +
        '.' +
        this.subnetIP3.num +
        '.' +
        this.subnetIP4.num;
    }
    if (idx === 'gate') {
      this.edgeGateway =
        this.gateIP1.num +
        '.' +
        this.gateIP2.num +
        '.' +
        this.gateIP3.num +
        '.' +
        this.gateIP4.num;
    }
    entity.error = false;
    this.invalidInput = false;
  }
  checkDuplicates(value) {
    if (
      this.edges.some(
        e => e.name.trim().toLowerCase() === value.trim().toLowerCase()
      )
    )
      this.duplicateEdgeFound = true;
    else this.duplicateEdgeFound = false;
  }
}
