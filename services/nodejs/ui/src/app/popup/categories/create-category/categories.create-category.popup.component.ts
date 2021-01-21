import { Component } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { Category } from '../../../model/category';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';
import { element } from 'protractor';
import { TableBaseComponent } from '../../../base-components/table.base.component';

interface ParamMetadata {
  key: string;
  val: string;
}

function newParamMetadata() {
  return {
    key: uuidv4(),
    val: '',
  };
}

@Component({
  selector: 'app-categories-create-category-popup',
  templateUrl: './categories.create-category.popup.component.html',
  styleUrls: ['./categories.create-category.popup.component.css'],
})
export class CategoriesCreateCategoryPopupComponent extends TableBaseComponent {
  categoryName = '';
  categoryPurpose = '';
  categoryValues = [];
  isConfirmLoading = false;
  queryParamSub = null;
  categoryId = null;
  category: Category = null;
  showAddParamTable = false;
  editRow: string = null;
  tempEditObject: any = {};
  dataSources = [];
  dataStreams = [];
  categories = [];
  isLoading = false;
  hasDuplicateName = false;
  hasDuplicateValue = false;
  categoryNameCopy = '';
  categoryAction = '';
  viewModal = false;
  duplicateValueKey = '';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private regService: RegistryService,
    private http: Http
  ) {
    super(router);
    this.fetchData();
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.id) {
        this.showAddParamTable = true;
        // id param exists - update case
        let cat = this.regService.get(params.id);
        if (cat) {
          this.categoryId = cat.id;
          this.category = cat;
          this.categoryName = cat.name;
          this.categoryNameCopy = cat.name;
          this.categoryPurpose = cat.purpose;
          this.categoryAction = cat.action;
          this.categoryValues = cat.values.map(v => ({
            val: v,
            key: uuidv4(),
          }));
          if (this.categoryAction === 'duplicate') this.categoryName = '';
        } else this.refresh(params.id);
      }
    });
  }
  async fetchData() {
    this.isLoading = true;
    try {
      this.dataSources = await this.http
        .get('/v1/datasources', getHttpRequestOptions())
        .toPromise()
        .then(
          x => x.json(),
          e => {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchData()
            );
          }
        );
      this.dataStreams = await this.http
        .get('/v1/datastreams', getHttpRequestOptions())
        .toPromise()
        .then(
          x => x.json(),
          e => {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchData()
            );
          }
        );
      this.categories = await this.http
        .get('/v1/categories', getHttpRequestOptions())
        .toPromise()
        .then(
          x => x.json(),
          e => {
            handleAuthError(null, e, this.router, this.http, () =>
              this.fetchData()
            );
          }
        );
      if (this.categoryAction !== 'duplicate') this.checkCategoryValues();
      else this.isLoading = false;
    } catch (e) {
      handleAuthError(null, e, this.router, this.http, () => this.fetchData());
      this.isLoading = false;
    }
  }

  async refresh(id) {
    try {
      const data = await this.http
        .get(`/v1/categories/${id}`, getHttpRequestOptions())
        .toPromise()
        .then(
          x => x.json(),
          e =>
            handleAuthError(null, e, this.router, this.http, () =>
              this.refresh(id)
            )
        );

      this.categoryId = data.id;
      this.category = data;
      this.categoryName = data.name;
      this.categoryNameCopy = data.name;
      this.categoryPurpose = data.purpose;
      this.categoryValues = data.values.map(v => ({
        val: v,
        key: uuidv4(),
      }));
    } catch (e) {}
  }
  checkCategoryValues() {
    this.viewModal = false;
    this.categoryValues.forEach(val => {
      this.dataSources.forEach(ds => {
        if (ds.selectors.find(c => c.value === val.val)) {
          val.disabled = true;
        }
      });
      this.dataStreams.forEach(dst => {
        if (
          !!dst.originSelectors &&
          dst.originSelectors.find(c => c.value === val.val)
        ) {
          val.disabled = true;
        }
      });

      if (val.disabled) this.viewModal = true;
    });
    this.isLoading = false;
  }
  onClosePopup() {
    if (this.categoryId) {
      this.regService.register(this.categoryId, null);
    }
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onCreateCategory() {
    this.isConfirmLoading = true;
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    const id = uuidv4();

    const cat = {
      id,
      tenantId,
      name: this.categoryName,
      purpose: this.categoryPurpose,
      values: this.categoryValues.map(c => c.val),
    };
    let method = 'post';
    if (this.category !== null && this.categoryAction !== 'duplicate') {
      cat['id'] = this.category.id;
      method = 'put';
    }
    this.isConfirmLoading = true;
    this.http[method]('/v1/categories', cat, getHttpRequestOptions())
      .toPromise()
      .then(
        r => {
          this.isConfirmLoading = false;
          this.router.navigate([{ outlets: { popup: null } }]);
        },
        err => {
          this.isConfirmLoading = false;
          const text =
            this.categoryAction === 'duplicate'
              ? 'clone'
              : method === 'post' ? 'create' : 'update';
          const warning = 'Failed to ' + text + ' category';
          this.isConfirmLoading = false;
          this.router.navigate([{ outlets: { popup: null } }]);
          handleAuthError(
            () => alert(warning),
            err,
            this.router,
            this.http,
            () => this.onCreateCategory()
          );
        }
      );
  }

  isCreateDisabled() {
    return (
      this.hasDuplicateName ||
      this.hasDuplicateValue ||
      !this.categoryName ||
      this.categoryValues.length === 0 ||
      this.categoryValues[0].val.length === 0
    );
  }
  onClickAddParam() {
    this.showAddParamTable = true;
    if (this.categoryValues.length === 0) {
      this.onAddParam();
    }
  }
  onAddParam() {
    const pm = newParamMetadata();

    if (
      this.hasDuplicateValue ||
      (this.categoryValues[0] && this.categoryValues[0].val.length === 0)
    ) {
    } else {
      this.categoryValues.unshift(pm);
      this.tempEditObject[pm.key] = pm;
      this.editRow = pm.key;
    }
  }
  edit(data) {
    this.tempEditObject[data.key] = { ...data };
    this.editRow = data.key;
  }

  save(event, data) {
    event.stopPropagation();
    Object.assign(data, this.tempEditObject[data.key]);
    this.editRow = null;
  }

  cancel(event, data) {
    event.stopPropagation();
    this.tempEditObject[data.key] = {};
    this.editRow = null;
    const idx = this.categoryValues.findIndex(v => v.key === data.key);
    if (data.key === this.duplicateValueKey) {
      this.hasDuplicateValue = false;
      this.duplicateValueKey = '';
    }
    if (idx !== -1) {
      this.categoryValues.splice(idx, 1);
      if (this.categoryValues.length === 0) {
        this.showAddParamTable = false;
      }
    }
  }

  clickRow(data) {
    if (this.hasDuplicateValue) {
      return;
    }
    if (this.editRow !== data.key) {
      this.edit(data);
    }
  }

  checkCategoryDuplicates(field, v) {
    if (field === 'name') {
      this.hasDuplicateName = false;
      const name = this.categoryName.trim().toLowerCase();
      const nameCopy = this.categoryNameCopy.trim().toLowerCase();
      if (name === '') {
        return;
      }
      this.categories.forEach(c => {
        const n = c.name.toLowerCase();
        if (n === name && n !== nameCopy) {
          this.hasDuplicateName = true;
          return;
        }
      });
    }

    if (field === 'value') {
      this.hasDuplicateValue = false;
      this.duplicateValueKey = '';
      const val = v.trim().toLowerCase();
      if (val === '') {
        return;
      }
      this.categoryValues.forEach(c => {
        const n = c.val.trim().toLowerCase();
        if (n === val && c.key !== this.editRow) {
          this.hasDuplicateValue = true;
          this.duplicateValueKey = this.editRow;
          return;
        }
      });
    }
  }
}
