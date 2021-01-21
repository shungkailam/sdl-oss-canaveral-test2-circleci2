import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

@Component({
  selector: 'app-project-runtime',
  templateUrl: './project.runtime.component.html',
  styleUrls: ['./project.runtime.component.css'],
})
export class ProjectRuntimeComponent extends TableBaseComponent {
  columns = ['Name', 'Image Path', 'Description'];
  data = [];
  isLoading = false;
  queryParamSub = null;
  projectId = '';
  routerEventUrl = '';
  isConfirmLoading = false;
  isDeleteModalVisible = false;
  toDelete = [];
  numberOfBuiltin = 0;
  isModalConfirmLoading = false;

  sortMap = {
    Name: null,
    'Image Path': null,
    Description: null,
  };

  mapping = {
    Name: 'name',
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
    this.queryParamSub = this.route.parent.params.subscribe(params => {
      if (params && params.id) {
        this.projectId = params.id;
        this.routerEventUrl = `/project/${this.projectId}/runtime`;
      }
    });
  }

  async fetchData() {
    this.isLoading = true;
    this.http
      .get(
        `v1/projects/${this.projectId}/scriptruntimes`,
        getHttpRequestOptions()
      )
      .toPromise()
      .then(
        res => {
          const data = res.json();
          data.forEach(r => {
            if (!r.builtin && !r.projectId) r.global = true;
            if (r.global && this._sherlockRole === '') r.viewonly = true;
            if (r.builtin) r.viewonly = true;
          });
          const builtInEle = data.filter(x => x.builtin);
          if (builtInEle && builtInEle.length) {
            this.numberOfBuiltin = builtInEle.length;
          }
          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
          this.isLoading = false;
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchData()
          );
          this.isLoading = false;
        }
      );
  }

  onCreateRuntime() {
    this.router.navigate(
      [{ outlets: { popup: ['scripts', 'create-runtime'] } }],
      {
        queryParams: { projectId: this.projectId },
        queryParamsHandling: 'merge',
      }
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
  ngOnDestroy() {
    this.queryParamSub.unsubscribe();
    super.ngOnDestroy();
    this.unsubscribeRouterEventMaybe();
  }
}
