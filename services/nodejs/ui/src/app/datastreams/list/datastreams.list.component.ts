import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../base-components/table.base.component';
import { RegistryService } from '../../services/registry.service';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';

@Component({
  selector: 'app-datastreams-list',
  templateUrl: './datastreams.list.component.html',
  styleUrls: ['./datastreams.list.component.css'],
})
export class DataStreamsListComponent extends TableBaseComponent {
  columns = [
    'Name',
    'Project',
    'Origin',
    'Destination',
    'Stream Type',
    // 'Size',
  ];

  data = [];
  isConfirmLoading = false;

  // subscribe to router event for create datastream
  routerEventUrl = '/datastreams/list';

  sortMap = {
    Name: null,
    Project: null,
    Origin: null,
    Destination: null,
    'Stream Type': null,
  };

  // to resolve the naming conflict between the table title and the key from table data source
  mapping = {
    Name: 'name',
    Project: 'project',
    Origin: 'origin',
    Destination: 'destination',
    'Stream Type': 'streamType',
    Size: 'size',
  };

  isLoading = false;
  toDeletes = [];
  showConfirmDelete = false;
  sourceDS = [];
  disabledDS = [];
  showProjectModal = false;
  projects = [];
  projectId = '';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private registryService: RegistryService,
    private http: Http
  ) {
    super(router);
  }

  fetchData() {
    this.isLoading = true;
    let promise = [];
    promise.push(
      this.http.get('/v1/datastreams', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/users', getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 3) {
          const data = res[0].json();
          const projects = res[1].json();
          const users = res[2].json();
          this.sourceDS = [];
          this.projects = [];
          const user = users.find(
            u => u.email.trim() === this._sherlockUsername
          );
          if (user) {
            projects.forEach(p => {
              if (p.users && p.users.find(pu => pu.userId === user.id)) {
                this.projects.push(p);
              }
            });
          }
          data.forEach(d => {
            if (d.origin === 'Data Stream') {
              this.sourceDS.push(d.originId);
            }
            this.projects.forEach(p => {
              if (p.id === d.projectId) d.project = p.name;
            });
          });
          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.getStreamType();
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
    this.showProjectModal = true;
  }

  selectProject(isSelected) {
    this.showProjectModal = false;
    if (!isSelected) {
      return;
    }
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
      this.showConfirmDelete = false;
      this.isConfirmLoading = false;
      return;
    }
    const promises = this.toDeletes.map(c =>
      this.http
        .delete(`/v1/datastreams/${c.id}`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.fetchData();
        this.isConfirmLoading = false;
        this.showConfirmDelete = false;
      },
      err => {
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
    this._rowIndex = '';
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
}
