import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { reject } from '../../../../../node_modules/@types/q';

@Component({
  selector: 'app-project-datastreams',
  templateUrl: './project.datastreams.component.html',
  styleUrls: ['./project.datastreams.component.css'],
})
export class ProjectDatastreamsComponent extends TableBaseComponent {
  columns = [
    'Name',
    'Origin',
    'Destination',
    'Stream Type',
    // 'Size',
  ];

  data = [];
  queryParamSub = null;
  projectId = '';
  routerEventUrl = '';
  isConfirmLoading = false;
  toDeletes = [];
  showConfirmDelete = false;
  disabledDS = [];
  sourceDS = [];
  isModalConfirmLoading = false;

  sortMap = {
    Name: null,
    Origin: null,
    Destination: null,
    'Stream Type': null,
    Size: null,
  };

  mapping = {
    Name: 'name',
    Origin: 'origin',
    Destination: 'destination',
    'Stream Type': 'streamType',
    Size: 'size',
  };

  isLoading = false;

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private registryService: RegistryService,
    private http: Http
  ) {
    super(router);
    this.queryParamSub = this.route.parent.params.subscribe(params => {
      if (params && params.id) {
        this.projectId = params.id;
        this.routerEventUrl = `/project/${this.projectId}/datastreams`;
      }
    });
  }

  fetchData() {
    this.isLoading = true;
    try {
      this.http
        .get(
          `/v1/projects/${this.projectId}/datastreams`,
          getHttpRequestOptions()
        )
        .toPromise()
        .then(
          x => {
            const data = x.json();
            if (data) {
              data.sort((a, b) => a.name.localeCompare(b.name));
              this.data = data;
              this.getStreamType();
              this.sourceDS = [];
              this.data.forEach(d => {
                if (d.origin === 'Data Stream') {
                  this.sourceDS.push(d.originId);
                }
              });
            }
            this.isLoading = false;
          },
          reject => {
            handleAuthError(null, reject, this.router, this.http, () =>
              this.fetchData()
            );
            this.isLoading = false;
          }
        );
    } catch (e) {
      handleAuthError(null, e, this.router, this.http, () => this.fetchData());
      this.isLoading = false;
    }
  }

  getStreamType() {
    this.data.forEach(d => {
      if (d.destination === 'Edge') {
        d.streamType =
          d.edgeStreamType === 'None' ? 'RealTime' : d.edgeStreamType;
      } else {
        d.streamType =
          d.cloudType === 'AWS' ? d.awsStreamType : d.gcpStreamType;
      }
    });
  }

  onClickEntity(entity) {
    // this.router.navigate(['project', project._id]);
    alert('clicked data stream ' + entity.name);
  }

  onClickCreateDataStream() {
    this.router.navigate(
      [{ outlets: { popup: ['datastreams', 'create-datastream'] } }],
      {
        queryParams: { projectId: this.projectId },
        queryParamsHandling: 'merge',
      }
    );
  }

  onClickRemoveTableRow() {
    this.isConfirmLoading = true;
    this.toDeletes = [];
    this.showConfirmDelete = true;
    if (this._rowIndex) {
      this.toDeletes = this._displayData.filter(x => x.id === this._rowIndex);
    } else {
      this.toDeletes = this._displayData.filter(x => x.checked);
    }

    this.disabledDS = [];
    this.toDeletes.forEach(d => {
      const item = this.sourceDS.find(s => s === d.id);
      if (item) {
        this.disabledDS.push(d);
      }
    });
    this._rowIndex = '';
  }

  confirmDelete(confirmDelete) {
    if (!confirmDelete) {
      this.isConfirmLoading = false;
      this.isModalConfirmLoading = false;
      this.showConfirmDelete = false;
      return;
    }
    this.isModalConfirmLoading = true;
    const promises = this.toDeletes.map(c =>
      this.http
        .delete(`/v1/datastreams/${c.id}`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.fetchData();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.showConfirmDelete = false;
      },
      err => {
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.showConfirmDelete = false;
        handleAuthError(
          () => alert('Failed to delete datastreams'),
          err,
          this.router,
          this.http,
          () => this.confirmDelete(confirmDelete)
        );
      }
    );
  }

  onClickUpdateTableRow() {
    const ds = this._displayData.find(d => d.id === this._rowIndex);
    console.log('>>> update, item=', ds);
    this.registryService.register(ds.id, ds);
    this.router.navigate(
      [{ outlets: { popup: ['datastreams', 'create-datastream'] } }],
      { queryParams: { id: ds.id }, queryParamsHandling: 'merge' }
    );
  }

  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      // const data = this.data;
      // // don't allow delete if a selected stream is the origin of some stream
      // return !this._displayData.some(
      //   d => d.checked && data.some(dt => dt.originId === d.id)
      // );
      return true;
    }
    return false;
  }
  ngOnDestroy() {
    this.queryParamSub.unsubscribe();
    super.ngOnDestroy();
    this.unsubscribeRouterEventMaybe();
  }
}
