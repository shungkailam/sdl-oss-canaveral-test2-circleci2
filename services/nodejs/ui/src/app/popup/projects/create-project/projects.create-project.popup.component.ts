import { Component, OnInit, OnDestroy, ViewChild } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { TableBaseComponent } from '../../../base-components/table.base.component';

import { Project, Category } from '../../../model/index';
import * as omit from 'object.omit';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { datasourceMatchOriginSelectors } from '../../../utils/modelUtil';
import { handleAuthError } from '../../../utils/authUtil';

@Component({
  selector: 'app-projects-create-project-popup',
  templateUrl: './projects.create-project.popup.component.html',
  styleUrls: ['./projects.create-project.popup.component.css'],
})
export class ProjectsCreateProjectPopupComponent extends TableBaseComponent
  implements OnInit, OnDestroy {
  isConfirmLoading = false;
  projectName = '';
  projectDescription = '';
  openModal = false;
  selectionType = '';
  // which step are we on in the create flow
  current = 0;
  searchVal = '';

  // sensor table columns
  columns = ['Edge Name'];
  matchedEdges = [];

  // all categories for tenant
  categories: Category[] = [];
  edges = [];
  users = [];
  selectedUsers = [];
  selectedUsersCopy = [];
  selectAllUsers = false;
  allUsers = [];
  selectedEdges = [];
  selectedEdgesCopy = [];
  selectAllEdges = false;
  allEdges = [];
  clouds = [];
  containers = [];
  dataSources = [];
  edgeSelectionType = 'category';
  projectId = '';
  catInfos = [
    {
      id: '',
      value: '',
      values: [],
    },
  ];
  cloudInfo = [
    {
      id: '',
      val: '',
    },
  ];
  containerInfo = [
    {
      id: '',
      val: '',
    },
  ];
  isUpdate = false;
  queryParamSub = null;
  project = null;
  allProjects = [];
  duplicateNameFound = false;

  // all edges for tenant

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.id) {
        this.isUpdate = true;
        let project = this.regService.get(params.id);
        this.projectId = params.id;
        this.project = project;
        if (project) {
          this.projectName = project.name;
          this.projectDescription = project.description;
          this.edgeSelectionType =
            project.edgeSelectorType === 'Category' ? 'category' : 'edge';
        }
      }
    });
    this.fetchClouds();
    this.fetchContainers();
    this.fetchEdges();
    this.fetchCategories();
    this.fetchUsers();
    this.fetchProjects();
  }

  ngOnInit() {
    super.ngOnInit();
    // subscribe to query param to see if we are within an edge context, then set edgeId, else fetch all edges
    this.queryParamSub = this.route.queryParams.subscribe(params => {});
    this.fetchDataSources();
  }

  fetchContainers() {
    return this.http
      .get('/v1/containerregistries', getHttpRequestOptions())
      .toPromise()
      .then(
        cs => {
          this.containers = cs.json();
          this.containers.forEach(c => {
            c.selected = false;
          });
          if (
            this.isUpdate &&
            this.project &&
            this.project.dockerProfileIds &&
            this.project.dockerProfileIds.length > 0
          ) {
            this.containerInfo = [];
            this.project.dockerProfileIds.forEach(e => {
              const containerItem = this.containers.find(c => c.id === e);
              if (containerItem) {
                this.containerInfo.push({
                  id: containerItem.id,
                  val: containerItem.name,
                });
              }
            });
          }
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchContainers()
          );
        }
      );
  }

  fetchDataSources() {
    return this.http
      .get('/v1/datasources', getHttpRequestOptions())
      .toPromise()
      .then(
        cs => {
          this.dataSources = cs.json();
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchDataSources()
          );
        }
      );
  }

  fetchCategories() {
    return this.http
      .get('/v1/categories', getHttpRequestOptions())
      .toPromise()
      .then(
        cs => {
          this.categories = cs.json();
          if (
            this.isUpdate &&
            this.edgeSelectionType === 'category' &&
            this.project &&
            this.project.edgeSelectors &&
            this.project.edgeSelectors.length > 0
          ) {
            this.catInfos = [];
            this.project.edgeSelectors.forEach(e => {
              const oldVal = e.value;
              this.onChangeSelectCategory(e);
              e.value = oldVal;
              this.catInfos.push(e);
            });
            let callingList = [];
            callingList.push(
              this.http.get('/v1/edges', getHttpRequestOptions()).toPromise()
            );
            callingList.push(
              this.http
                .get('/v1/datasources', getHttpRequestOptions())
                .toPromise()
            );
            Promise.all(callingList).then(
              res => {
                if (res[0] && res[1]) {
                  this.edges = res[0].json();
                  this.dataSources = res[1].json();
                  this.updateAffectedEdge('category');
                }
              },
              rej => {
                handleAuthError(null, rej, this.router, this.http, () =>
                  this.fetchCategories()
                );
              }
            );
          }
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchCategories()
          );
        }
      );
  }

  fetchEdges() {
    return this.http
      .get('/v1/edges', getHttpRequestOptions())
      .toPromise()
      .then(
        es => {
          this.edges = es.json();
          this.edges.forEach(ele => {
            ele.selected = false;
          });
          this.allEdges = this.edges.slice();
          this.selectedEdges = [];
          if (
            this.isUpdate &&
            this.project &&
            this.project.edgeIds &&
            this.project.edgeIds.length > 0
          ) {
            this.selectedEdgesCopy = [];
            this.matchedEdges = [];
            this.project.edgeIds.forEach(e => {
              const edgeItem = this.allEdges.find(ae => ae.id === e);
              if (edgeItem) {
                edgeItem.selected = true;
                this.selectedEdgesCopy.push(edgeItem);
                this.selectedEdges.push(edgeItem);
                this.matchedEdges.push(edgeItem);
                this.selectAllEdges =
                  this.selectedEdges.length === this.allEdges.length;
              }
            });
          }
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchEdges()
          );
        }
      );
  }

  fetchUsers() {
    return this.http
      .get('/v1/users', getHttpRequestOptions())
      .toPromise()
      .then(
        es => {
          this.users = es.json();
          this.users.forEach(ele => {
            ele.selected = false;
          });
          this.allUsers = this.users.slice();
          this.selectedUsers = [];
          this.selectedUsersCopy = [];
          if (
            this.isUpdate &&
            this.project &&
            this.project.users &&
            this.project.users.length > 0
          ) {
            this.project.users.forEach(u => {
              const uItem = this.allUsers.find(au => u.userId === au.id);
              if (uItem) {
                uItem.selected = true;
                this.selectedUsersCopy.push(uItem);
                this.selectedUsers.push(uItem);
                this.selectAllUsers =
                  this.selectedUsers.length === this.allUsers.length;
              }
            });
          }
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchUsers()
          );
        }
      );
  }

  fetchClouds() {
    return this.http
      .get('/v1/cloudcreds', getHttpRequestOptions())
      .toPromise()
      .then(
        es => {
          this.clouds = es.json();
          this.clouds.forEach(c => {
            c.selected = false;
          });
          if (
            this.isUpdate &&
            this.project &&
            this.project.cloudCredentialIds &&
            this.project.cloudCredentialIds.length > 0
          ) {
            this.cloudInfo = [];
            this.project.cloudCredentialIds.forEach(e => {
              const cloudItem = this.clouds.find(c => c.id === e);
              if (cloudItem) {
                this.cloudInfo.push({
                  id: cloudItem.id,
                  val: cloudItem.name,
                });
              }
            });
          }
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchClouds()
          );
        }
      );
  }

  fetchProjects() {
    this.http
      .get('/v1/projects', getHttpRequestOptions())
      .toPromise()
      .then(
        res => {
          this.allProjects = res.json();
        },
        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchProjects()
          );
        }
      );
  }

  checkDuplicateName(name) {
    const n = name.toLowerCase().trim();
    var originName = '';
    if (this.project) {
      originName = this.project.name;
    }
    this.duplicateNameFound = false;
    this.allProjects.forEach(ap => {
      const an = ap.name.toLowerCase().trim();
      if (n === an && n !== originName) {
        this.duplicateNameFound = true;
        return;
      }
    });
  }

  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onCreateProject() {
    this.isConfirmLoading = true;
    const projectObj = this.handleProjectObj();
    let method = this.isUpdate ? 'put' : 'post';
    this.http[method]('/v1/projects', projectObj, getHttpRequestOptions())
      .toPromise()
      .then(
        response => {
          this.isConfirmLoading = false;
          this.onClosePopup();
        },
        rej => {
          this.isConfirmLoading = false;
          const warning =
            'Failed to ' +
            (method === 'post' ? 'create' : 'update') +
            ' project';
          handleAuthError(
            () => alert(warning),
            rej,
            this.router,
            this.http,
            () => this.onCreateProject()
          );
        }
      );
  }

  handleProjectObj() {
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    let cloudCredentialIds = [];
    this.cloudInfo.forEach(co => {
      if (co.id !== '') {
        cloudCredentialIds.push(co.id);
      }
    });
    let dockerProfileIds = [];
    this.containerInfo.forEach(co => {
      if (co.id !== '') {
        dockerProfileIds.push(co.id);
      }
    });
    let edgeSelectorType = '';
    if (this.edgeSelectionType === 'edge') {
      edgeSelectorType = 'Explicit';
      var edgeIds = [];
      this.matchedEdges.forEach(m => {
        edgeIds.push(m.id);
      });
    } else {
      edgeSelectorType = 'Category';
      var edgeSelectors = [];
      this.catInfos.forEach(ca => {
        if (ca.id !== '') {
          let newItem = {
            id: ca.id,
            value: ca.value,
          };
          edgeSelectors.push(newItem);
        }
      });
    }
    let users = [];
    this.selectedUsersCopy.forEach(s => {
      let userItem = {
        role: s.role,
        userId: s.id,
      };
      users.push(userItem);
    });
    let projectObj = {
      cloudCredentialIds,
      description: this.projectDescription,
      dockerProfileIds,
      edgeSelectorType,
      id: uuidv4(),
      name: this.projectName,
      tenantId,
      users,
    };
    if (this.edgeSelectionType === 'edge') {
      projectObj['edgeIds'] = edgeIds;
    } else {
      projectObj['edgeSelectors'] = edgeSelectors;
    }
    if (this.isUpdate) {
      projectObj['id'] = this.project.id;
    }
    return projectObj;
  }

  //
  isCreateDisabled() {}

  onNext() {
    this.current += 1;
  }

  onBack() {
    this.current -= 1;
  }

  // whether Next button should be disabled or not
  isNextDisabled() {
    return (
      !this.projectName ||
      !this.projectName.trim() ||
      this.selectedUsersCopy.length === 0 ||
      this.duplicateNameFound
    );
  }

  onOpenItemSelection(type) {
    this.openModal = true;
    this.selectionType = type;
  }

  selectItems(isConfirmed) {
    this.openModal = false;
    if (isConfirmed && this.selectionType === 'User') {
      this.selectedUsersCopy = this.selectedUsers.slice();
    }
    if (isConfirmed && this.selectionType === 'Edge') {
      this.selectedEdgesCopy = this.selectedEdges.slice();
      this.updateAffectedEdge('edge');
    }
  }

  onSelectAllItems = function() {
    if (this.selectionType === 'User') {
      if (this.selectAllUsers) {
        this.users.forEach(ele => {
          ele.selected = true;
        });
        this.selectedUsers = this.users.slice();
      } else {
        this.users.forEach(ele => {
          ele.selected = false;
        });
        this.selectedUsers = [];
      }
    } else {
      if (this.selectAllEdges) {
        this.edges.forEach(ele => {
          ele.selected = true;
        });
        this.selectedEdges = this.edges.slice();
      } else {
        this.edges.forEach(ele => {
          ele.selected = false;
        });
        this.selectedEdges = [];
      }
    }
  };

  onSelectUser = function(user) {
    if (user.selected) {
      this.selectedUsers.push(user);
    } else {
      let userIndex = -1;
      this.selectedUsers.some(function(ele, index) {
        if (ele.id === user.id) {
          userIndex = index;
          return;
        }
      });
      if (userIndex >= 0) {
        this.selectedUsers.splice(userIndex, 1);
      }
    }
    this.selectAllUsers = this.selectedUsers.length === this.allUsers.length;
  };

  removeUser = function(user) {
    user.selected = false;
    this.onSelectUser(user);
    this.selectedUsersCopy = this.selectedUsers.slice();
    let changedUser = this.users.find(u => u.id === user.id);
    changedUser.selected = false;
  };

  onSelectEdge = function(edge) {
    if (edge.selected) {
      this.selectedEdges.push(edge);
    } else {
      let edgeIndex = -1;
      this.selectedEdges.some(function(ele, index) {
        if (ele.id === edge.id) {
          edgeIndex = index;
          return;
        }
      });
      if (edgeIndex >= 0) {
        this.selectedEdges.splice(edgeIndex, 1);
      }
    }
    this.selectAllEdges = this.selectedEdges.length === this.allEdges.length;
  };

  removeEdge = function(edge) {
    edge.selected = false;
    this.onSelectEdge(edge);
    this.selectedEdgesCopy = this.selectedEdges.slice();
    let changedEdge = this.edges.find(u => u.id === edge.id);
    changedEdge.selected = false;
    this.updateAffectedEdge('edge');
  };

  onFilterChange() {
    const searchVal = this.searchVal.trim().toLowerCase();
    const newItems = [];
    if (this.selectionType === 'User') {
      this.allUsers.forEach(e => {
        const name = e.name.toLowerCase();
        if (searchVal.length === 0 || name.indexOf(searchVal) > -1) {
          newItems.push(e);
        }
      });
      this.users = [];
      this.selectedUsers = [];
      for (let i = 0; i < this.allUsers.length; i++) {
        for (let j = 0; j < newItems.length; j++) {
          if (this.allUsers[i].id === newItems[j].id) {
            this.users.push(this.allUsers[i]);
            if (this.allUsers[i].selected) {
              this.selectedUsers.push(this.allUsers[i]);
            }
            break;
          }
        }
      }
    } else {
      this.allEdges.forEach(e => {
        const name = e.name.toLowerCase();
        if (searchVal.length === 0 || name.indexOf(searchVal) > -1) {
          newItems.push(e);
        }
      });
      this.edges = [];
      this.selectedEdges = [];
      for (let i = 0; i < this.allEdges.length; i++) {
        for (let j = 0; j < newItems.length; j++) {
          if (this.allEdges[i].id === newItems[j].id) {
            this.edges.push(this.allEdges[i]);
            if (this.allEdges[i].selected) {
              this.selectedEdges.push(this.allEdges[i]);
            }
            break;
          }
        }
      }
    }
  }

  updateAffectedEdge(type) {
    this.matchedEdges = [];
    if (type === 'edge') {
      this.matchedEdges = this.selectedEdgesCopy.slice();
    }
    if (type === 'category') {
      const dss = this.dataSources.filter(ds =>
        datasourceMatchOriginSelectors(ds, this.catInfos, '')
      );
      if (dss.length) {
        let edgeMap = {};
        dss.forEach(ds => {
          const edge = this.edges.find(e => e.id === ds.edgeId);
          if (edge && !edgeMap[edge.id]) {
            edgeMap[edge.id] = true;
            this.matchedEdges.push(edge);
          }
        });
      } else {
        this.matchedEdges = [];
      }
    }
  }

  onChangeSelectCategory(ci) {
    const cat = this.categories.find(c => c.id === ci.id);
    if (cat) {
      ci.values = cat.values;
      ci.value = '';
    }
  }

  onClickAddCategoryInfo() {
    this.catInfos.push({
      id: '',
      value: '',
      values: [],
    });
  }

  onClickCloseCategoryInfo(i) {
    this.catInfos.splice(i, 1);
  }

  changeCloud() {
    this.clouds.forEach(c => {
      c.selected = false;
    });
    this.cloudInfo.forEach(c => {
      let cloudItem = this.clouds.find(cl => cl.id === c.id);
      if (cloudItem) {
        cloudItem.selected = true;
      }
    });
    console.log(this.cloudInfo);
  }

  onClickAddCloud() {
    this.cloudInfo.push({
      id: '',
      val: '',
    });
  }

  onClickCloseCloudInfo(index) {
    this.cloudInfo.splice(index, 1);
    this.changeCloud();
  }

  onClickAddContainer() {
    this.containerInfo.push({
      id: '',
      val: '',
    });
  }

  changeContainer() {
    this.containers.forEach(c => {
      c.selected = false;
    });

    this.containerInfo.forEach(c => {
      let containerItem = this.containers.find(cl => cl.id === c.id);
      if (containerItem) {
        containerItem.selected = true;
      }
    });
    console.log(this.containerInfo);
  }

  onClickCloseContainerInfo(index) {
    this.containerInfo.splice(index, 1);
    this.changeContainer();
  }
}
