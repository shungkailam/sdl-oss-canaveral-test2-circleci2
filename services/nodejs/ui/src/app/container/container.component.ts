import { Component } from '@angular/core';
import {
  Router,
  ActivatedRoute,
  ParamMap,
  NavigationEnd,
} from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../base-components/table.base.component';
import { RegistryService } from '../services/registry.service';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';

@Component({
  selector: 'app-settings',
  templateUrl: './container.component.html',
  styleUrls: ['./container.component.css'],
})
export class ContainerComponent extends TableBaseComponent {
  data = [];
  isLoading = false;
  columns = ['Name', 'Container Registry', 'Description'];

  isConfirmLoading = false;
  // subscribe to router event for create profile
  routerEventUrl = '/container';

  sortMap = {
    Name: null,
    'Container Registry': null,
    Description: null,
  };

  mapping = {
    Name: 'name',
    'Container Registry': 'type',
    Description: 'description',
  };

  isDeleteModalVisible = false;
  isModalConfirmLoading = false;
  alertClosed = false;
  viewModal = false;
  multipleProfiles = false;
  associatedProfiles = [];
  toDelete = [];

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private registryService: RegistryService,
    private http: Http
  ) {
    super(router);
    this.data = [];
  }
  async fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http
        .get('/v1/containerregistries', getHttpRequestOptions())
        .toPromise()
    );
    promise.push(
      this.http.get('/v1/scriptruntimes', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 3) {
          const data = res[0].json();
          const runtimeData = res[1].json();
          const projectsData = res[2].json();
          data.forEach(dp => {
            if (dp.type === 'ContainerRegistry')
              data.type = 'Container Registry';
            if (runtimeData.some(sr => sr.dockerProfileID === dp.id))
              dp.disable = true;
            projectsData.forEach(p => {
              if (p.dockerProfileIds) {
                if (p.dockerProfileIds.some(pId => pId === dp.id))
                  dp.disable = true;
              }
            });
          });
          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.isLoading = false;
        }
      },
      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }

  onClickCreateRegistryProfile() {
    this.router.navigate(
      [{ outlets: { popup: ['container', 'create-container'] } }],
      { queryParamsHandling: 'merge' }
    );
  }
  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      return true;
    }
    return false;
  }

  updateRowIndex(id, option) {
    this._rowIndex = id;
    if (option.label === 'Edit') this.onClickUpdateTableRow();
    if (option.label === 'Remove') this.onClickRemoveTableRow();
    if (option.label === 'View') this.onClickViewTableRow();
  }

  onClickRemoveTableRow() {
    this.isConfirmLoading = true;
    this.isDeleteModalVisible = true;
    this.toDelete = [];
    this.associatedProfiles = [];
    this.isLoading = true;
    this.viewModal = false;
    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDelete = this._displayData.filter(x => x.checked);

    if (this.toDelete.length === 1) this.multipleProfiles = false;
    else this.multipleProfiles = true;

    this.toDelete.forEach(d => {
      if (d.disable) {
        this.viewModal = true;
        this.associatedProfiles.push(d);
      }
    });
    this._rowIndex = '';
    this.isLoading = false;
  }

  doDeleteProfile() {
    const promises = this.toDelete.map(p =>
      this.http
        .delete(`/v1/containerregistries/${p.id}`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.fetchData();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
      },
      err => {
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = true;
        this.isDeleteModalVisible = false;
        handleAuthError(
          () => alert('Failed to delete docker profile'),
          err,
          this.router,
          this.http,
          () => this.doDeleteProfile()
        );
      }
    );
  }
  onClickUpdateTableRow() {
    const profile = this._displayData.find(rp => rp.id === this._rowIndex);

    console.log('>>> update, item=', profile);
    this.registryService.register(profile.id, profile);
    this.router.navigate(
      [{ outlets: { popup: ['container', 'create-container'] } }],
      { queryParams: { id: profile.id }, queryParamsHandling: 'merge' }
    );
    this._rowIndex = '';
  }
  onClickViewTableRow() {
    const profile = this._displayData.find(rp => rp.id === this._rowIndex);
    this.registryService.register(profile.id, profile);
    this.router.navigate(
      [{ outlets: { popup: ['container', 'create-container'] } }],
      { queryParams: { id: profile.id }, queryParamsHandling: 'merge' }
    );
    this._rowIndex = '';
  }
  handleDeleteProfileOk() {
    this.isModalConfirmLoading = true;
    this.doDeleteProfile();
  }
  handleDeleteProfileCancel() {
    this.isConfirmLoading = false;
    this.isDeleteModalVisible = false;
  }
  onCloseAlert() {
    this.alertClosed = true;
  }
  ngOnDestroy() {
    this.unsubscribeRouterEventMaybe();
  }
}
