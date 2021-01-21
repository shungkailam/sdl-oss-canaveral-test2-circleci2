import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { AggregateInfo } from '../../../model/index';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

@Component({
  selector: 'app-project-applications',
  templateUrl: './project.applications.component.html',
  styleUrls: ['./project.applications.component.css'],
})
export class ProjectApplicationsComponent extends TableBaseComponent {
  isLoading = false;
  isDeleteModalVisible = false;
  toDelete = [];
  data = [];
  columns = ['Name', 'Description', 'Last Updated'];
  queryParamSub = null;
  projectId = '';
  routerEventUrl = '';
  isConfirmLoading = false;
  project = '';
  isModalConfirmLoading = false;

  sortMap = {
    Name: null,
    Description: null,
    'Last Updated': null,
  };

  mapping = {
    Name: 'name',
    Description: 'description',
    'Last Updated': 'last_updated',
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
        this.routerEventUrl = `/project/${this.projectId}/applications`;
      }
    });
  }
  fetchData() {
    this.isLoading = true;
    this.http
      .get(
        `/v1/projects/${this.projectId}/applications`,
        getHttpRequestOptions()
      )
      .toPromise()
      .then(
        response => {
          this.data = response.json();
          this.data.forEach(e => {
            const date = new Date(e.updatedAt);
            const time = date.toLocaleString();
            e.last_updated = time;
          });
          this.isLoading = false;
        },
        reject => {
          handleAuthError(null, reject, this.router, this.http, () =>
            this.fetchData()
          );
          this.isLoading = false;
        }
      );
  }
  onClickCreateApplications() {
    this.router.navigate(
      [{ outlets: { popup: ['applications', 'create-application'] } }],
      {
        queryParams: { projectId: this.projectId },
        queryParamsHandling: 'merge',
      }
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

  onClickRemoveTableRow() {
    this.isDeleteModalVisible = true;
    this.isConfirmLoading = true;
    if (this._allChecked) {
      this.toDelete = this._displayData.filter(x => x.checked);
    } else if (this._indeterminate) {
      this.toDelete = this._displayData.filter(x => x.checked);
    } else {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    }
  }

  onClickUpdateTableRow() {
    const appl = this._displayData.find(appl => appl.id === this._rowIndex);
    this.regService.register(appl.id, appl);
    this.router.navigate(
      [{ outlets: { popup: ['applications', 'create-application'] } }],
      {
        queryParams: { id: appl.id },
        queryParamsHandling: 'merge',
      }
    );
  }

  OnClickDeleteApplication = function(isDelete) {
    this.isModalConfirmLoading = true;
    let deleteList = [];
    if (isDelete) {
      this.toDelete.forEach(d => {
        deleteList.push(
          this.http
            .delete(`/v1/application/${d.id}`, getHttpRequestOptions())
            .toPromise()
        );
      });
      Promise.all(deleteList).then(
        () => {
          this.isModalConfirmLoading = false;
          this.isConfirmLoading = false;
          this.isDeleteModalVisible = false;
          this.deleteItems = [];
          this.getApplications();
        },
        rej => {
          this.isModalConfirmLoading = false;
          this.isConfirmLoading = false;
          this.isDeleteModalVisible = false;
          handleAuthError(
            () => alert('Failed to delete application'),
            rej,
            this.router,
            this.http,
            () => this.OnClickDeleteApplication(isDelete)
          );
        }
      );
    } else {
      this.isDeleteModalVisible = false;
      this.deleteItems = [];
    }
  };
  showAppDetails(entity) {
    this.regService.register(entity.id, entity);
    this.router.navigate(['application', entity.id], {
      queryParamsHandling: 'merge',
    });
  }
  ngOnDestroy() {
    this.queryParamSub.unsubscribe();
    super.ngOnDestroy();
    this.unsubscribeRouterEventMaybe();
  }
  OnClickDeleteApplicationCancel() {
    this.isConfirmLoading = false;
    this.isDeleteModalVisible = false;
  }
}
