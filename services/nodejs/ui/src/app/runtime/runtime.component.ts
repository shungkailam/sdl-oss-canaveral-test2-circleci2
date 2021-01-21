import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../base-components/table.base.component';
import { RegistryService } from '../services/registry.service';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';

@Component({
  selector: 'app--runtime',
  templateUrl: './runtime.component.html',
  styleUrls: ['./runtime.component.css'],
})
export class ScriptsRuntimeComponent extends TableBaseComponent {
  columns = ['Name', 'Project', 'Image Path', 'Description'];
  data = [];
  isLoading = false;
  routerEventUrl = '/runtime';
  isDeleteModalVisible = false;
  toDelete = [];
  numberOfBuiltin = 0;
  projectsData = [];
  isConfirmLoading = false;
  isModalConfirmLoading = false;
  systemRuntimes = [];
  systemRuntimesLength = 0;
  sortMap = {
    Name: null,
    Project: null,
    'Image Path': null,
    Description: null,
  };

  mapping = {
    Name: 'name',
    Project: 'project',
    'Image Path': 'dockerRepoURI',
    Description: 'description',
  };

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
  }

  async fetchData() {
    this.isLoading = true;
    let promise = [];
    let runtimes = [];
    this.systemRuntimes = [];

    promise.push(
      this.http.get('/v1/scriptruntimes', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/users', getHttpRequestOptions()).toPromise()
    );
    Promise.all(promise).then(
      response => {
        if (response.length === 3) {
          const data = response[0].json();
          const projects = response[1].json();
          const users = response[2].json();
          const builtInEle = data.filter(x => x.builtin);
          if (builtInEle && builtInEle.length) {
            this.numberOfBuiltin = builtInEle.length;
          }
          const currentUser = users.find(
            u => u.email.trim() === this._sherlockUsername
          );
          if (currentUser) {
            projects.forEach(p => {
              if (p.users && p.users.find(pu => pu.userId === currentUser.id)) {
                this.projectsData.push(p);
              }
            });
          }
          data.forEach(r => {
            if (!r.builtin && !r.projectId) {
              r.global = true;
              this.systemRuntimes.push(r);
            }
            if (r.global && this._sherlockRole === '') r.viewonly = true;
            if (r.builtin) {
              r.viewonly = true;
              this.systemRuntimes.push(r);
            }
            this.projectsData.forEach(p => {
              if (p.id === r.projectId) r.project = p.name;
            });
          });
          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.systemRuntimesLength = data.length - this.systemRuntimes.length;
          this.isLoading = false;
        }
      },
      reject => {
        handleAuthError(null, reject, this.router, this.http, () =>
          this.fetchData()
        );
        this.isLoading = false;
      }
    );
  }

  onCreateRuntime() {
    this.router.navigate(
      [{ outlets: { popup: ['scripts', 'create-runtime'] } }],
      { queryParamsHandling: 'merge' }
    );
  }

  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length > this.numberOfBuiltin
    ) {
      return true;
    }
    return false;
  }

  onClickRemoveTableRow() {
    this.isDeleteModalVisible = true;
    this.isConfirmLoading = true;
    this.toDelete = [];

    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else
      this.toDelete = this._displayData.filter(x => x.checked && !x.builtin);

    this._rowIndex = '';
  }

  onClickUpdateTableRow() {
    const runtime = this._displayData.find(
      runtime => runtime.id === this._rowIndex
    );
    this.regService.register(runtime.id, runtime);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['scripts', 'create-runtime'] } }],
      {
        queryParams: { id: runtime.id },
        queryParamsHandling: 'merge',
      }
    );
  }
  onClickViewTableRow() {
    const runtime = this._displayData.find(
      runtime => runtime.id === this._rowIndex
    );
    this.regService.register(runtime.id, runtime);
    this._rowIndex = '';
    this.router.navigate(
      [{ outlets: { popup: ['scripts', 'create-runtime'] } }],
      {
        queryParams: { id: runtime.id },
        queryParamsHandling: 'merge',
      }
    );
  }

  OnClickDeleteRuntime(canDelete) {
    this.isModalConfirmLoading = true;
    if (!canDelete) {
      this.isDeleteModalVisible = false;
      return;
    }
    let deleteList = [];
    this.toDelete.forEach(d => {
      deleteList.push(
        this.http
          .delete(`/v1/scriptruntimes/${d.id}`, getHttpRequestOptions())
          .toPromise()
      );
    });
    Promise.all(deleteList).then(
      res => {
        this.fetchData();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
      },
      rej => {
        this.fetchData();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
        handleAuthError(
          () => alert('Failed to delete runtime'),
          rej,
          this.router,
          this.http,
          () => this.OnClickDeleteRuntime(canDelete)
        );
      }
    );
  }
  OnClickDeleteRuntimeCancel() {
    this.isModalConfirmLoading = false;
    this.isConfirmLoading = false;
    this.isDeleteModalVisible = false;
  }
}
