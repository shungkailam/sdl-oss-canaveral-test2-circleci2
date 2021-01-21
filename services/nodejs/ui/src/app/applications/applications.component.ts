import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import { TableBaseComponent } from '../base-components/table.base.component';
import { AggregateInfo } from '../model/index';
import { RegistryService } from '../services/registry.service';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';
import { Edge } from '../model/index';
import { reject } from 'q';

@Component({
  selector: 'app-applications',
  templateUrl: './applications.component.html',
  styleUrls: ['./applications.component.css'],
})
export class ApplicationsComponent extends TableBaseComponent {
  isLoading = false;
  isDeleteModalVisible = false;
  toDelete = [];
  data = [];
  columns = ['Name', 'Project', 'Description', 'Last Updated'];
  routerEventUrl = '/applications';
  projects = [];

  sortMap = {
    Name: null,
    Description: null,
    'Last Updated': null,
  };
  isConfirmLoading = false;
  isModalConfirmLoading = false;

  mapping = {
    Name: 'name',
    Project: 'project',
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
  }
//to pull latest version of UI(359)
  async fetchData() {
    this.isLoading = true;
    this.getApplications();
  }

  getApplications() {
    this.isLoading = true;
    let promise = [];
    let applications = [];
    promise.push(
      this.http.get('/v1/applications', getHttpRequestOptions()).toPromise()
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
          const currentUser = users.find(
            u => u.email.trim() === this._sherlockUsername
          );
          if (currentUser) {
            projects.forEach(p => {
              if (p.users && p.users.find(pu => pu.userId === currentUser.id)) {
                this.projects.push(p);
              }
            });
          }
          data.forEach(d => {
            const date = new Date(d.updatedAt);
            const time = date.toLocaleString();
            d.last_updated = time;
            this.projects.forEach(p => {
              if (p.id === d.projectId) d.project = p.name;
            });
          });

          data.sort((a, b) => a.name.localeCompare(b.name));
          this.data = data;
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

  onClickCreateApplications() {
    this.router.navigate(
      [{ outlets: { popup: ['applications', 'create-application'] } }],
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

  onClickRemoveTableRow() {
    this.isConfirmLoading = true;
    this.isDeleteModalVisible = true;
    this.toDelete = [];
    if (this._rowIndex) {
      this.toDelete = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDelete = this._displayData.filter(x => x.checked);
    this._rowIndex = '';
  }

  onClickUpdateTableRow() {
    const appl = this._displayData.find(appl => appl.id === this._rowIndex);
    this.regService.register(appl.id, appl);
    this._rowIndex = '';
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
        res => {
          this.getApplications();
          this.isModalConfirmLoading = false;
          this.isConfirmLoading = false;
          this.isDeleteModalVisible = false;
          this.deleteItems = [];
        },
        rej => {
          this.isModalConfirmLoading = false;
          this.isConfirmLoading = false;
          this.isDeleteModalVisible = false;
          handleAuthError(
            () => {
              alert('Failed to delete applications');
            },
            rej,
            this.router,
            this.http,
            () => this.OnClickDeleteApplication(isDelete)
          );
        }
      );
    } else {
      this.isConfirmLoading = false;
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
  handleCancelDeleteApp() {
    this.isModalConfirmLoading = false;
    this.isConfirmLoading = false;
  }
}
