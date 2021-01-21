import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import * as uuidv4 from 'uuid/v4';
import { User } from '../../../model/index';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

@Component({
  selector: 'app-project-users',
  templateUrl: './project.users.component.html',
  styleUrls: ['./project.users.component.css'],
})
export class ProjectUsersComponent extends TableBaseComponent {
  columns = ['Name', 'Email'];
  data = [];
  queryParamSub = null;
  projectId = '';
  routerEventUrl = '';
  isConfirmLoading = false;
  isCreateModalVisible = false;
  usernameInput = '';
  emailInput = '';
  passwordInput = '';
  userToUpdate: User = null;
  currentUser = '';
  viewModal = false;
  duplicateUserNameFound = false;
  duplicateUserEmailFound = false;
  updateUser = false;
  multipleUsers = false;
  isLoading = false;
  isDeleteModalVisible = false;
  associatedUsers = [];
  userRole = '';
  alertClosed = false;
  isModalConfirmLoading = false;
  showEmailError = false;

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService
  ) {
    super(router);
    this.queryParamSub = this.route.parent.params.subscribe(params => {
      if (params && params.id) {
        this.projectId = params.id;
        this.routerEventUrl = `/project/${this.projectId}/users`;
      }
    });
  }
  async fetchData() {
    this.isLoading = true;
    try {
      await this.http
        .get(`v1/projects/${this.projectId}/users`, getHttpRequestOptions())
        .toPromise()
        .then(
          response => {
            const data = response.json();
            this.currentUser = this._sherlockUsername;

            data.forEach(p => {
              p.storageStatus = 'OK';
              p.computeStatus = 'OK';

              if (
                this.currentUser.trim().toLowerCase() ===
                p.email.trim().toLowerCase()
              )
                p.disable = true;
            });

            this.data = data;
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

  updateRowIndex(id, option) {
    this._rowIndex = id;
    if (option.label === 'Edit') this.onClickUpdateTableRow();
    if (option.label === 'Remove') this.onClickRemoveTableRow();
    if (option.label === 'View') this.onClickViewTableRow();
  }

  onClickUpdateTableRow() {
    const user: User = this._displayData.find(u => u.id === this._rowIndex);
    this.updateUser = true;
    if (user['disable']) this.viewModal = true;
    else this.viewModal = false;

    console.log('>>> update, item=', user);
    this.userToUpdate = user;
    this.isCreateModalVisible = true;
    this.usernameInput = user.name;
    this.emailInput = user.email;
    this.userRole = user.role;
    // we only store password hash, so update requires re-enter password
    this.passwordInput = '';
    this._rowIndex = '';
  }
  onClickViewTableRow() {
    const user: User = this._displayData.find(u => u.id === this._rowIndex);
    this.viewModal = true;
    this.updateUser = true;
    console.log('>>> update, item=', user);
    this.userToUpdate = user;
    this.isCreateModalVisible = true;
    this.usernameInput = user.name;
    this.emailInput = user.email;
    this.userRole = user.role;
    // we only store password hash, so update requires re-enter password
    this.passwordInput = '';
    this._rowIndex = '';
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

  handleCreateUserCancel() {
    this.isCreateModalVisible = false;
    this.userToUpdate = null;
    this.updateUser = false;
    this.viewModal = false;
  }

  handleCreateUserOk() {
    this.isModalConfirmLoading = true;
    const tenantId = this.registryService.get(REG_KEY_TENANT_ID);
    let id = uuidv4();
    let method = 'post';
    if (this.userToUpdate) {
      id = this.userToUpdate.id;
      this.userToUpdate = null;
      method = 'put';
    }
    const user = {
      name: this.usernameInput,
      email: this.emailInput,
      password: this.passwordInput,
      role: this.userRole,
      id,
      tenantId,
    };
    this.http[method]('/v1/users', user, getHttpRequestOptions())
      .toPromise()
      .then(
        r => {
          this.fetchData();
          this.updateUser = false;
          this.isModalConfirmLoading = false;
          this.isCreateModalVisible = false;
          this.viewModal = false;
        },
        err => {
          this.fetchData();
          this.isModalConfirmLoading = false;
          const warning =
            'Failed to ' + (method === 'post' ? 'create' : 'update') + ' user';
          handleAuthError(
            () => alert(warning),
            err,
            this.router,
            this.http,
            () => this.handleCreateUserOk()
          );
        }
      );
  }

  checkDuplicateUserNames(entity) {
    if (entity === 'name') {
      if (
        this.data.some(
          u =>
            u.name.trim().toLowerCase() ===
            this.usernameInput.trim().toLowerCase()
        )
      )
        this.duplicateUserNameFound = true;
      else this.duplicateUserNameFound = false;
    }
    if (entity === 'email') {
      if (
        this.data.some(
          u =>
            u.email.trim().toLowerCase() ===
            this.emailInput.trim().toLocaleLowerCase()
        )
      )
        this.duplicateUserEmailFound = true;
      else this.duplicateUserEmailFound = false;
    }
  }
  userEmailChange() {
    const re = /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    if (re.test(this.emailInput)) {
      this.showEmailError = false;
    } else {
      this.showEmailError = true;
    }
  }

  createUserDisabled() {
    return (
      !this.usernameInput ||
      !this.emailInput ||
      !this.passwordInput ||
      !this.userRole ||
      this.duplicateUserEmailFound ||
      this.duplicateUserNameFound ||
      this.showEmailError
    );
  }

  onCloseAlert() {
    this.alertClosed = true;
  }
  ngOnDestroy() {
    this.queryParamSub.unsubscribe();
    super.ngOnDestroy();
    this.unsubscribeRouterEventMaybe();
  }
}
