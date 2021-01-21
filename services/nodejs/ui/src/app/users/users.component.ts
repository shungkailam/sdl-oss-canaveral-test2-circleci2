import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../services/registry.service';
import { TableBaseComponent } from '../base-components/table.base.component';
import * as uuidv4 from 'uuid/v4';
import { User } from '../model/index';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';

@Component({
  selector: 'app-users',
  templateUrl: './users.component.html',
  styleUrls: ['./users.component.css'],
})
export class UsersComponent extends TableBaseComponent {
  columns = ['Name', 'Email'];
  data = [];
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
  showEmailError = false;
  alertClosed = false;
  toDeletes = [];
  userRole = 'INFRA_ADMIN';
  sherlockRole = '';
  isModalConfirmLoading = false;

  isLoading = false;
  isDeleteModalVisible = false;
  associatedUsers = [];
  projectsData = [];
  routerEventUrl = '/users';
  isInviteModalVisible = false;
  isInfra = false;
  isProjectUser = false;
  inviteUserRole = [];
  inviteUsersName = '';
  inviteUsersEmail = '';
  duplicateInviteUserEmailFound = false;
  duplicateInviteUserNameFound = false;
  showInviteEmailError = false;
  projects = [];
  selectedProjects = [];
  selectAllProjects = false;
  allProjects = [];
  searchVal = '';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private registryService: RegistryService
  ) {
    super(router);
    this.getProjects();
  }
  async fetchData() {
    this.isLoading = true;
    try {
      const data = await this.http
        .get('/v1/users', getHttpRequestOptions())
        .toPromise()
        .then(
          response => response.json(),
          rej =>
            handleAuthError(null, rej, this.router, this.http, () =>
              this.fetchData()
            )
        );
      this.projectsData = await this.http
        .get('/v1/projects', getHttpRequestOptions())
        .toPromise()
        .then(
          response => response.json(),
          rej =>
            handleAuthError(null, rej, this.router, this.http, () =>
              this.fetchData()
            )
        );

      this.currentUser = this._sherlockUsername;
      if (this.updateUser) {
        this._totalProjects = 0;
        this.projectsData.forEach(p => {
          if (p.users) {
            p.users.forEach(pUser => {
              data.forEach(u => {
                if (u.id === pUser.userId) {
                  if (
                    u.email.trim().toLowerCase() ===
                    this._sherlockUsername.trim().toLowerCase()
                  ) {
                    if (u.role === 'USER') {
                      localStorage.removeItem('sherlock_role');
                      localStorage['sherlock_role'] = '';
                      this.router.navigate(['/']);
                    }
                    this.isModalConfirmLoading = false;
                    this.isCreateModalVisible = false;
                    this.viewModal = false;
                  }
                }
              });
            });
          }
        });
      }

      this.updateUser = false;
      data.forEach(p => {
        p.storageStatus = 'OK';
        p.computeStatus = 'OK';

        if (
          this.currentUser.trim().toLowerCase() === p.email.trim().toLowerCase()
        )
          p.disable = true;
      });
      data.sort((a, b) => a.name.localeCompare(b.name));
      this.data = data;

      this.isLoading = false;
    } catch (e) {
      handleAuthError(null, e, this.router, this.http, () => this.fetchData());
      this.isLoading = false;
    }
  }

  getProjects() {
    this.http
      .get('/v1/projects', getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          this.projects = response.json();
          this.projects.forEach(ele => {
            ele.selected = false;
          });
          this.allProjects = this.projects.slice();
          this.selectedProjects = [];
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.getProjects()
          );
        }
      );
  }

  onClickEntity(user) {
    // this.router.navigate(['user', user.id]);
  }

  onClickCreateUser() {
    this.isCreateModalVisible = true;
    this.usernameInput = '';
    this.emailInput = '';
    this.passwordInput = '';
    this.duplicateUserEmailFound = false;
    this.duplicateUserNameFound = false;
  }
  updateRowIndex(id, option) {
    this._rowIndex = id;
    if (option.label === 'Edit') this.onClickUpdateTableRow();
    if (option.label === 'Remove') this.onClickRemoveTableRow();
    if (option.label === 'View') this.onClickViewTableRow();
  }

  onClickRemoveTableRow() {
    this.toDeletes = [];
    this.associatedUsers = [];
    this.isDeleteModalVisible = true;
    this.viewModal = false;
    this.isLoading = true;
    this.isConfirmLoading = true;

    if (this._rowIndex) {
      this.toDeletes = this._displayData.filter(x => x.id === this._rowIndex);
    } else this.toDeletes = this._displayData.filter(x => x.checked);

    if (this.toDeletes.length === 1) this.multipleUsers = false;
    else this.multipleUsers = true;

    this.toDeletes.forEach(d => {
      if (d.disable) {
        this.viewModal = true;
        this.associatedUsers.push(d);
      }
    });

    this._rowIndex = '';
    this.isLoading = false;
  }

  doDeleteUser() {
    const promises = this.toDeletes.map(c =>
      this.http.delete(`/v1/users/${c.id}`, getHttpRequestOptions()).toPromise()
    );
    Promise.all(promises).then(
      () => {
        this.fetchData();
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
      },
      err => {
        handleAuthError(
          () => alert('Failed to delete user'),
          err,
          this.router,
          this.http,
          () => this.doDeleteUser()
        );
        this.isModalConfirmLoading = false;
        this.isConfirmLoading = false;
        this.isDeleteModalVisible = false;
      }
    );
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
    if (user['disable'] || this._sherlockRole === '') this.viewModal = true;
    else this.viewModal = false;
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
          if (this._sherlockRole !== '' && this.viewModal) {
            if (localStorage['sherlock_creds']) {
              const obj = JSON.parse(localStorage['sherlock_creds']);
              const username = obj['username'];
              localStorage['sherlock_creds'] = JSON.stringify({
                username,
                password: this.passwordInput,
              });
            }
          }
          if (!this.updateUser || user.email !== this.currentUser) {
            this.isModalConfirmLoading = false;
            this.isCreateModalVisible = false;
            this.viewModal = false;
          }
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

  handleDeleteUserOk() {
    this.isModalConfirmLoading = true;
    this.doDeleteUser();
  }
  handleDeleteUserCancel() {
    this.isConfirmLoading = false;
    this.isDeleteModalVisible = false;
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

  disableUserSelection() {
    if (this._sherlockRole === '') {
      return true;
    }
    let numberOfAdmin = 0,
      isCurrentUser = false;
    this.data.forEach(d => {
      if (d.role === 'INFRA_ADMIN') {
        numberOfAdmin++;
      }
      if (
        this.emailInput &&
        this.emailInput.trim() === this._sherlockUsername
      ) {
        isCurrentUser = true;
      }
    });
    return numberOfAdmin === 1 && isCurrentUser;
  }

  onClickInviteUser(showModal) {
    this.isInviteModalVisible = showModal;
    if (!showModal) {
      return;
    }
  }

  addUserRole() {
    this.inviteUserRole = [];
    if (this.isInfra) {
      this.inviteUserRole.push('infra');
    }
    if (this.isProjectUser) {
      this.inviteUserRole.push('project_user');
    }
  }

  checkDuplicateInviteUser(entity) {
    if (entity === 'name') {
      if (
        this.data.some(
          u =>
            u.name.trim().toLowerCase() ===
            this.inviteUsersName.trim().toLowerCase()
        )
      )
        this.duplicateInviteUserNameFound = true;
      else this.duplicateInviteUserNameFound = false;
    }
    if (entity === 'email') {
      if (
        this.data.some(
          u =>
            u.email.trim().toLowerCase() ===
            this.inviteUsersEmail.trim().toLowerCase()
        )
      )
        this.duplicateInviteUserEmailFound = true;
      else this.duplicateInviteUserEmailFound = false;
    }
  }

  checkInviteUserValid() {
    const re = /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    if (re.test(this.inviteUsersEmail)) {
      this.showInviteEmailError = false;
    } else {
      this.showInviteEmailError = true;
    }
  }

  onSelectAllProjects = function() {
    if (this.selectAllProjects) {
      this.projects.forEach(ele => {
        ele.selected = true;
      });
      this.selectedProjects = this.projects.slice();
    } else {
      this.projects.forEach(ele => {
        ele.selected = false;
      });
      this.selectedProjects = [];
    }
  };

  onSelectProject = function(project) {
    if (project.selected) {
      this.selectedProjects.push(project);
    } else {
      let projectIndex = -1;
      this.selectedProjects.some(function(ele, index) {
        if (ele.id === project.id) {
          projectIndex = index;
          return;
        }
      });
      if (projectIndex >= 0) {
        this.selectedProjects.splice(projectIndex, 1);
      }
    }
    this.selectAllProjects =
      this.selectedProjects.length === this.allProjects.length;
  };

  onFilterChange() {
    const searchVal = this.searchVal.trim().toLowerCase();
    const newItems = [];
    this.allProjects.forEach(e => {
      const name = e.name.toLowerCase();
      if (searchVal.length === 0 || name.indexOf(searchVal) > -1) {
        newItems.push(e);
      }
    });
    this.projects = [];
    this.selectedProjects = [];
    for (let i = 0; i < this.allProjects.length; i++) {
      for (let j = 0; j < newItems.length; j++) {
        if (this.allProjects[i].id === newItems[j].id) {
          this.projects.push(this.allProjects[i]);
          if (this.allProjects[i].selected) {
            this.selectedProjects.push(this.allProjects[i]);
          }
          break;
        }
      }
    }
  }

  disableInvite() {
    return (
      (!this.isInfra && !this.isProjectUser) ||
      this.inviteUsersName === '' ||
      this.inviteUsersEmail === '' ||
      this.duplicateInviteUserEmailFound ||
      this.duplicateInviteUserNameFound ||
      this.showInviteEmailError
    );
  }
}
