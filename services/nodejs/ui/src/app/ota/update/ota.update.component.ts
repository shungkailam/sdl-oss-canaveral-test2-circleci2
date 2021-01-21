import { Component } from '@angular/core';
import { Router, ActivatedRoute, NavigationEnd } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../base-components/table.base.component';
import { RegistryService } from '../../services/registry.service';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';
import * as uuidv4 from 'uuid/v4';

@Component({
  selector: 'app-ota-update',
  templateUrl: './ota.update.component.html',
  styleUrls: ['./ota.update.component.css'],
})
export class OtaUpdateComponent extends TableBaseComponent {
  isLoading = false;
  updates = [
    {
      name: 'Edge Updates',
      selected: false,
    },
  ];
  edgeInfo = [];
  edges = [];
  edgeUpdateVersion = '';
  updateAll = false;
  updateNone = true;
  updateTitle = '';
  showEdges = false;
  selectedEdges = [];
  selectAllEdges = false;
  selectedEntitiesText = 'No Entities';
  extraText = '';
  selectedEdgeNum = 0;
  routerEventSubscribe = null;
  routerEventUrl = '/ota/update';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.routerEventSubscribe = this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        if (event.url === '/ota/update' || event.url === '/ota') {
          this.getEdgeInfo();
        }
      }
    });
  }

  getEdgeInfo() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http
        .get('/v1/edgesCompatibleUpgrades', getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/edgesInfo', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      x => {
        if (x.length === 3) {
          const updateVersion = x[0].json();
          this.selectAllEdges = false;
          this.selectedEdgeNum = 0;
          if (updateVersion.length > 0) {
            this.edgeUpdateVersion = updateVersion[0]['release'];
            this.edgeInfo = x[1]
              .json()
              .filter(e => e.EdgeVersion !== this.edgeUpdateVersion);
            const edges = x[2].json();
            this.edges = [];
            edges.forEach(e => {
              const edgeItem = this.edgeInfo.find(eI => eI.edgeId === e.id);
              if (edgeItem) {
                e.selected = false;
                e.edgeVersion = edgeItem.EdgeVersion;
                this.edges.push(e);
              }
            });
          }
        }
        this.isLoading = false;
      },
      e => {
        handleAuthError(null, e, this.router, this.http, () =>
          this.getEdgeInfo()
        );
        this.isLoading = false;
      }
    );
  }

  onClickUpdate() {
    if (this.updateNone) {
      this.updates.forEach(u => {
        u.selected = true;
      });
    }
    const updateId = uuidv4();
    let updateInfo = {};
    updateInfo['edge'] = this.selectedEdges;
    updateInfo['updateVersion'] = this.edgeUpdateVersion;
    updateInfo['updates'] = this.updates;
    this.regService.register(updateId, updateInfo);
    this.router.navigate([{ outlets: { popup: ['ota', 'confirm-update'] } }], {
      queryParams: { id: updateId },
      queryParamsHandling: 'merge',
    });
    console.log('update all');
  }

  changeUpdateSelection() {
    if (this.selectedEdges.length === 0) {
      return;
    }
    let updateNum = 0;
    this.updates.forEach(u => {
      if (u.selected) {
        updateNum++;
      }
    });
    this.updateAll = updateNum === this.updates.length;
    this.updateNone = updateNum === 0;
  }

  editEntities() {
    console.log('Edit entities');
    this.showEdges = true;
    this.updateTitle =
      'Select edges for Edge Update - ' + this.edgeUpdateVersion + ' update.';
  }

  OnClickSelectEdge(selected) {
    if (!selected) {
      this.showEdges = false;
      return;
    }
    this.selectedEdges = [];
    this.edges.forEach(e => {
      if (e.selected) {
        this.selectedEdges.push(e);
      }
    });
    if (this.selectedEdges.length > 0) {
      this.selectedEntitiesText = 'Edge ' + this.selectedEdges[0].name;
    } else {
      this.selectedEntitiesText = 'No Entities';
    }

    if (this.selectedEdges.length > 1) {
      this.extraText = ' and ' + (this.selectedEdges.length - 1) + ' more';
    } else {
      this.extraText = '';
    }
    let updateNum = 0;
    this.updates.forEach(u => {
      if (u.selected) {
        updateNum++;
      }
    });
    this.updateAll = updateNum === this.updates.length;
    this.updateNone = updateNum === 0;
    this.showEdges = false;
  }

  selectAllEdge() {
    this.edges.forEach(e => {
      e.selected = this.selectAllEdges;
    });
    if (this.selectAllEdges) {
      this.selectedEdgeNum = this.edges.length;
    } else {
      this.selectedEdgeNum = 0;
    }
  }

  selectEdge(edge) {
    if (edge.selected) {
      this.selectedEdgeNum++;
    } else {
      this.selectedEdgeNum--;
    }
    this.selectAllEdges = this.selectedEdgeNum === this.edges.length;
  }

  ngOnDestroy() {
    this.routerEventSubscribe.unsubscribe();
    super.ngOnDestroy();
  }
}
