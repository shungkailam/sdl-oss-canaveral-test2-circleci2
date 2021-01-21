import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { TableBaseComponent } from '../../../base-components/table.base.component';

import {
  DataSourceFieldInfo,
  DataSource,
  Category,
  DATA_TYPES,
} from '../../../model/index';
import * as omit from 'object.omit';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

function convertRTSPTopic(
  topic: string,
  username: string,
  password: string
): string {
  topic = 'rtsp://' + topic.substr(topic.indexOf('@'), topic.length);
  if (topic.indexOf('rtsp://') === 0) {
    const prefix = `rtsp://${username}:${password}`;
    if (topic.indexOf(prefix) !== 0) {
      const suffix = topic.substring('rtsp://'.length);
      return `${prefix}${suffix}`;
    }
  }
  return topic;
}
function stripUsernamePasswordFromRTSP(topic: string): string {
  if (topic.indexOf('rtsp://') === 0) {
    const idx = topic.indexOf('@');
    if (idx !== -1) {
      const suffix = topic.substring(idx + 1);
      return `rtsp://${suffix}`;
    }
  }
  return topic;
}
const RE_RTSP = /rtsp:\/\/([^:@]+):([^@]+)@/;
function extractUsernamePasswordFromRTSP(rtspTopic: string) {
  const m = rtspTopic.match(RE_RTSP);
  if (m) {
    return {
      username: m[1],
      password: m[2],
    };
  }
  return null;
}
interface KeyedDataSourceFieldInfo extends DataSourceFieldInfo {
  key: string;
}
@Component({
  selector: 'app-datasources-create-datasource-popup',
  templateUrl: './datasources.create-datasource.popup.component.html',
  styleUrls: ['./datasources.create-datasource.popup.component.css'],
})
export class DataSourcesCreateDataSourcePopupComponent
  extends TableBaseComponent
  implements OnInit, OnDestroy {
  dataTypes = DATA_TYPES;
  dataSourceName = '';
  isConfirmLoading = false;

  // which step are we on in the create flow
  current = 0;

  // Sensor or Gateway
  sensorType: 'Sensor' | 'Gateway' = 'Sensor';

  // hardcoded list for .NEXT Nice
  sensorModel = 'Model S';

  // sensor table columns
  columns = ['Name', 'MQTT Topic'];

  connection: 'Secure' | 'Unsecure' = 'Unsecure';

  // all fields added
  fields: KeyedDataSourceFieldInfo[] = [];

  // all categories for tenant
  categories: Category[] = [];

  // all attributes added
  attributes = [];

  // current active attribute (used in Select Fields dialog)
  currentAttr = null;

  queryParamSub = null;
  0;
  edgeId = '';

  // all edges for tenant
  edges = [];

  // sensors for the currently selected edge
  sensors = [];

  // whether to show 'Select Fields' dialog
  isSelectFieldsVisible = false;

  // model for table in Select Fields dialog
  selectFieldComponent: TableBaseComponent = null;

  dataSource = null;

  sensorProtocol: 'MQTT' | 'RTSP' | 'OTHER' = 'RTSP';
  sensorAuth: 'PASSWORD' | 'CERTIFICATE' | 'TOKEN' = 'PASSWORD';

  username = '';
  password = '';

  generatingCertificate = false;
  generatedCertificate = false;

  editRow = '';

  tempEditObject: any = {};

  dataSourcesFields = {};
  datasourceFieldsNames = [];
  datasourceNames = [];
  editedTopic = '';
  duplicateTopicNameFound = false;
  duplicateFieldNameFound = false;
  editedFieldName = '';
  duplicateDatasourceNameFound = false;
  editedDatasoureName = '';
  isLoading = false;
  isUpdate = false;
  context = '';
  dsIPAddress = '';
  invalidInput = false;
  dsIP1 = { num: '', error: false };
  dsIP2 = { num: '', error: false };
  dsIP3 = { num: '', error: false };
  dsIP4 = { num: '', error: false };
  populatedUrl = '';
  editedRowId = '';
  currentIp = '';
  invalidTopic = false;
  currentUrl = '';

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
  }

  ngOnInit() {
    super.ngOnInit();
    this.fetchCategories();
    this.fecthDataSources();
    // subscribe to query param to see if we are within an edge context, then set edgeId, else fetch all edges
    this.queryParamSub = this.route.queryParams.subscribe(params => {
      if (params && params.projectId) {
        this.context = 'project';
      }
      if (params && params.id) {
        // id param exists - update case
        const ds: DataSource = this.regService.get(params.id);
        if (ds) {
          this.edgeId = ds.edgeId;
          this.dataSource = ds;
          this.dataSourceName = ds.name;
          this.sensorType = ds.type;
          this.editedDatasoureName = ds.name;
          this.sensorModel = ds.sensorModel;
          this.connection = ds.connection;
          this.fields = ds.fields.map(f => ({
            key: uuidv4(),
            ...f,
          }));
          this.sensorProtocol = ds.protocol;
          this.sensorAuth = ds.authType;

          if (this.sensorAuth === 'PASSWORD') {
            let idx = this.fields[0].mqttTopic.indexOf('@');

            if (idx > 0) {
              let ip = this.fields[0].mqttTopic.substr(
                this.fields[0].mqttTopic.indexOf('@') + 1
              );
              this.dsIPAddress = ip.substr(0, ip.indexOf('/'));
            } else {
              let ipAd = this.fields[0].mqttTopic.substr(7);
              this.dsIPAddress = ipAd.substr(0, ipAd.indexOf('/'));
            }

            let ips = this.dsIPAddress.split('.');
            this.dsIP1.num = ips[0];
            this.dsIP2.num = ips[1];
            this.dsIP3.num = ips[2];
            this.dsIP4.num = ips[3];
            if (idx > 0) {
              const m = extractUsernamePasswordFromRTSP(
                this.fields[0].mqttTopic
              );
              if (m) {
                this.username = m.username;
                this.password = m.password;
                let password = '';
                this.password.split('').forEach(p => {
                  password += '*';
                });
                this.fields.forEach(f => {
                  let topicItems = f.mqttTopic.split(':');
                  //let pswd = topicItems[2].substr(0, topicItems[2].indexOf('@'));
                  let hidePswd = topicItems[2].replace(this.password, password);
                  f.mqttTopic =
                    topicItems[0] + ':' + topicItems[1] + ':' + hidePswd;
                });
              }
            }
          }

          this.isUpdate = true;
          this.fetchCategories().then(x => {
            this.attributes = ds.selectors.map(s => {
              const cat = this.categories.find(c => c.id === s.id);
              const isAllScope =
                s.scope.length === 1 && s.scope[0] === '__ALL__';
              return {
                id: s.id,
                value: s.value,
                values: cat.values,
                scope: isAllScope
                  ? '__ALL__'
                  : this.fields.length ? 'selok' : 'sel',
                scopeFields: isAllScope ? [] : s.scope.slice(0),
                fields: this.cloneFields(),
              };
            });
          });
        }
      }
      if (!this.dataSource && params && params.edgeId) {
        this.edgeId = params.edgeId;
        this.isUpdate = true;
      }
      if (this.edgeId) {
        this.fetchEdgeSensors();
        this.fetchSpecificEdge();
      } else {
        // no edge id, load all edges
        this.fetchEdges();
      }
    });
    this.fields.forEach(f => {
      this.datasourceFieldsNames.push(f.name);
    });
  }
  ngOnDestroy() {
    if (this.queryParamSub) {
      this.queryParamSub.unsubscribe();
    }
    super.ngOnDestroy();
  }

  async fecthDataSources() {
    this.isLoading = true;
    this.http
      .get('/v1/datasources', getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          const dataSources = x.json();
          this.isLoading = false;
          dataSources.forEach(ds => {
            this.datasourceNames.push(ds.name.trim().toLowerCase());
            ds.fields.forEach(field => {
              if (!this.dataSourcesFields[ds.edgeId])
                this.dataSourcesFields[ds.edgeId] = [field.mqttTopic];
              else this.dataSourcesFields[ds.edgeId].push(field.mqttTopic);
            });
          });
        },
        e => {
          handleAuthError(null, e, this.router, this.http, () =>
            this.fecthDataSources()
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
        },
        err => {
          handleAuthError(null, err, this.router, this.http, () =>
            this.fetchCategories()
          );
        }
      );
  }

  fetchEdges() {
    this.http
      .get('/v1/edges', getHttpRequestOptions())
      .toPromise()
      .then(
        es => {
          this.edges = es.json();
          if (this.edges.length === 1) this.edgeId = this.edges[0].id;
        },
        err => {
          handleAuthError(null, err, this.router, this.http, () =>
            this.fetchEdges()
          );
        }
      );
  }

  fetchSpecificEdge() {
    this.http
      .get(`/v1/edges/${this.edgeId}`, getHttpRequestOptions())
      .toPromise()
      .then(
        es => {
          this.edges.push(es.json());
        },
        err => {
          handleAuthError(null, err, this.router, this.http, () =>
            this.fetchSpecificEdge()
          );
        }
      );
  }

  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onCreateDataSource() {
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    this.isConfirmLoading = true;
    const id = uuidv4();
    let fields = [];
    if (this.username === '') {
      fields = this.fields;
    } else {
      fields = this.fields.map(({ name, fieldType, mqttTopic }) => ({
        name,
        fieldType,
        mqttTopic: convertRTSPTopic(mqttTopic, this.username, this.password),
      }));
    }
    const ds: DataSource = {
      id,
      tenantId,
      edgeId: this.edgeId,
      name: this.dataSourceName,
      type: this.sensorType,
      sensorModel: this.sensorModel,
      connection: this.connection,
      fields: fields,
      selectors: this.attributes.map(a => ({
        id: a.id,
        value: a.value,
        scope: a.scope === '__ALL__' ? [a.scope] : a.scopeFields,
      })),
      protocol: this.sensorProtocol,
      authType: this.sensorAuth,
    };
    let method = 'post';
    if (this.dataSource) {
      ds['id'] = this.dataSource.id;
      method = 'put';
    }
    console.log(ds);
    this.http[method]('/v1/datasources', ds, getHttpRequestOptions())
      .toPromise()
      .then(
        r => {
          this.isConfirmLoading = false;
          this.router.navigate([{ outlets: { popup: null } }]);
        },
        x => {
          this.isConfirmLoading = false;
          const warning =
            'Failed to ' +
            (method === 'post' ? 'create' : 'update') +
            ' datasources';
          handleAuthError(() => alert(warning), x, this.router, this.http, () =>
            this.onCreateDataSource()
          );
          this.isConfirmLoading = false;
        }
      );
  }

  isCreateDisabled() {
    return this.attributes.filter(a => !a.id || !a.value).length !== 0;
  }

  onNext() {
    this.current += 1;
    if (this.current === 1) {
      if (this.sensorAuth === 'PASSWORD') {
        if (this.username === '') {
          this.populatedUrl = 'rtsp://' + this.dsIPAddress + '/';
        } else {
          let password = '';
          this.password.split('').forEach(p => {
            password += '*';
          });
          this.populatedUrl =
            'rtsp://' +
            this.username +
            ':' +
            password +
            '@' +
            this.dsIPAddress +
            '/';
        }

        if (this.currentUrl !== '' && this.currentUrl !== this.populatedUrl) {
          this.updateFields();
        }
        this.currentUrl = this.populatedUrl;
      }
    }

    // if step 3, sync up all fields
    if (this.current === 2) {
      this.checkEmptyFields();
      if (this.attributes.length === 0) {
        this.attributes.push(this.createAttributeTemplate());
      }

      this.attributes.forEach(a => {
        // first drop entries in scopeFields whose field no longer exists
        a.scopeFields = a.scopeFields.filter(f =>
          this.fields.find(fd => fd.name === f)
        );
        // update fields
        a.fields = this.cloneFields();
        // sync up field selection
        if (a.scopeFields.length) {
          a.fields.forEach(f => {
            f.checked = a.scopeFields.indexOf(f.name) !== -1;
          });
        }
      });
    }
  }
  checkEmptyFields() {
    this.fields.forEach((f, i) => {
      if (f.name === '' || f.mqttTopic === '') {
        this.fields.splice(i, 1);
      }
    });
  }
  onBack() {
    this.current -= 1;
  }

  updateFields() {
    this.fields.forEach(f => {
      let topic = f.mqttTopic.replace(this.currentUrl, this.populatedUrl);
      f.mqttTopic = topic;
    });
  }

  // handler for delete field
  onClickDeleteField() {
    const toDeletes = this._displayData.filter(x => x.checked);
    this._displayData = this._displayData.filter(x => !x.checked);
    this.fields = this.fields.filter(
      x => !toDeletes.find(y => y.name === x.name)
    );
    this._refreshStatus();
  }

  // whether Next button should be disabled or not
  isNextDisabled() {
    if (this.current === 0) {
      return (
        this.duplicateDatasourceNameFound ||
        !this.dataSourceName ||
        !this.edgeId ||
        ((this.sensorProtocol === 'RTSP' &&
          (!this.dsIP1.num ||
            !this.dsIP2.num ||
            !this.dsIP3.num ||
            !this.dsIP4.num)) ||
          (this.username && !this.password)) ||
        this.invalidInput
      );
    } else if (this.current === 1) {
      return !this.fields.filter(f => f.name && f.mqttTopic).length;
    }
    return true;
  }

  // selected category changed for attribute i
  onChangeSelectCategory(event, i) {
    const sc = this.categories.find(x => x.id === event.target.value);
    this.attributes[i].values = sc.values;
    this.attributes[i].value = sc.values[0];
  }

  // add attribute handler
  onClickAddAttribute() {
    this.attributes.push(this.createAttributeTemplate());
  }

  // delete attribute handler
  onDeleteAttribute(i) {
    this.attributes.splice(i, 1);
  }

  // edge selection change handler - load sensors for the selected edge
  onEdgeSelectionChange() {
    this.fetchEdgeSensors();
  }

  fetchEdgeSensors() {
    this.http
      .get(`/v1/edges/${this.edgeId}/sensors`, getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          this.sensors = x.json();
          console.log('>>> Got sensors:', this.sensors);
        },
        err => {
          handleAuthError(null, err, this.router, this.http, () =>
            this.fetchEdgeSensors()
          );
        }
      );
  }

  // attribute scope change handler
  onChangeScope(attr) {
    if (attr.scope === 'sel') {
      // pop up field selection dialog
      this.currentAttr = attr;
      this.isSelectFieldsVisible = true;
      this.selectFieldComponent = new TableBaseComponent(this.router);
    } else {
      this.selectFieldComponent = null;
    }
  }

  // select fields dialog cancel handler
  handleSelectFieldsCancel() {
    this.currentAttr.scope = '__ALL__';
    this.isSelectFieldsVisible = false;
  }

  // select fields dialog ok handler
  handleSelectFieldsOk() {
    this.isSelectFieldsVisible = false;
    this.currentAttr.scope = 'selok';
    // sync currentAttr.scopeFields with selection
    this.currentAttr.scopeFields = this.currentAttr.fields
      .filter(f => f.checked)
      .map(f => f.name);
  }

  // whether select fields dialog OK button should be disabled or not
  isSelectFieldsDisabled() {
    return (
      !this.currentAttr ||
      !this.currentAttr.fields ||
      this.currentAttr.fields.filter(f => f.checked).length === 0
    );
  }
  createAttributeTemplate() {
    return {
      // id of selected category
      id: '',
      // selected value in the selected category
      value: '',
      // all values in the selected category
      values: [],
      // scope for this attribute, __ALL__ or sel
      // sel means to select specific fields
      scope: '__ALL__',
      // if scope is sel, this stores selected field names
      scopeFields: [],
      // all fields available for selection
      // note: this may change if user enters step 3,
      // then back to step 2, add some more fields,
      // then enter step 3 again
      fields: this.cloneFields(),
    };
  }

  cloneFields() {
    return this.fields.map(d => ({ checked: false, ...omit(d, 'checked') }));
  }

  onClickGenerateCertificate() {
    this.generatingCertificate = true;
    setTimeout(() => {
      this.generatingCertificate = false;
      this.generatedCertificate = true;
    }, 5000);
  }

  onClickDownloadCertificate() {}

  onSensorProtocolChange() {
    if (this.sensorProtocol === 'RTSP') {
      this.sensorAuth = 'PASSWORD';
    } else if (this.sensorProtocol === 'MQTT') {
      this.sensorAuth = 'CERTIFICATE';
    }
  }

  onClickAddNewField() {
    if (this.fields[0] && (!this.fields[0].name || !this.fields[0].mqttTopic)) {
      return;
    }
    if (this.editedRowId) {
      if (
        this.tempEditObject[this.editedRowId] &&
        !this.tempEditObject[this.editedRowId].mqttTopic.includes(
          this.populatedUrl
        )
      ) {
        this.tempEditObject[this.editedRowId].mqttTopic =
          this.populatedUrl + this.tempEditObject[this.editedRowId].mqttTopic;
        if (this.fields.find(f => f.key === this.editedRowId))
          this.fields.find(
            f => f.key === this.editedRowId
          ).mqttTopic = this.tempEditObject[this.editedRowId].mqttTopic;
      }
    }
    const pm = {
      key: uuidv4(),
      name: '',
      fieldType: '',
      mqttTopic: '',
    };
    this.fields.unshift(pm);
    this.fields = [...this.fields];
    this.tempEditObject[pm.key] = { ...pm };
    this.editRow = pm.key;
  }

  edit(data) {
    if (this.editedRowId) {
      if (
        this.tempEditObject[this.editedRowId] &&
        !this.tempEditObject[this.editedRowId].mqttTopic.includes(
          this.populatedUrl
        )
      ) {
        this.tempEditObject[this.editedRowId].mqttTopic =
          this.populatedUrl + this.tempEditObject[this.editedRowId].mqttTopic;
        if (this.fields.find(f => f.key === this.editedRowId))
          this.fields.find(
            f => f.key === this.editedRowId
          ).mqttTopic = this.tempEditObject[this.editedRowId].mqttTopic;
      }
    }
    this.editedFieldName = data.name;
    this.duplicateTopicNameFound = false;
    this.duplicateFieldNameFound = false;
    this.invalidTopic = false;
    this.editedTopic = data.mqttTopic;
    data.mqttTopic = data.mqttTopic.substr(
      this.populatedUrl.length,
      data.mqttTopic.length
    );
    this.tempEditObject[data.key] = { ...data };
    this.editRow = data.key;
    this.editedRowId = data.key;
  }

  save(event, data) {
    event.stopPropagation();
    this.tempEditObject[data.key].mqttTopic =
      this.populatedUrl + this.tempEditObject[data.key].mqttTopic;

    let id = this.datasourceFieldsNames.indexOf(this.editedFieldName);
    if (id >= 0)
      this.datasourceFieldsNames[id] = this.tempEditObject[data.key].name;
    if (
      !this.datasourceFieldsNames.includes(this.tempEditObject[data.key].name)
    )
      this.datasourceFieldsNames.push(this.tempEditObject[data.key].name);

    if (this.dataSourcesFields[this.edgeId]) {
      let idx = this.dataSourcesFields[this.edgeId].indexOf(this.editedTopic);
      if (idx >= 0)
        this.dataSourcesFields[this.edgeId][idx] = this.tempEditObject[
          data.key
        ].mqttTopic;
    } else
      this.dataSourcesFields[this.edgeId] = [
        this.tempEditObject[data.key].mqttTopic,
      ];

    if (
      !this.dataSourcesFields[this.edgeId].includes(
        this.tempEditObject[data.key].mqttTopic
      )
    )
      this.dataSourcesFields[this.edgeId].push(
        this.tempEditObject[data.key].mqttTopic
      );

    Object.assign(data, this.tempEditObject[data.key]);
    this.editRow = null;
  }

  cancel(event, data) {
    if (this.datasourceFieldsNames.includes(data.name.trim().toLowerCase())) {
      const id = this.datasourceFieldsNames.indexOf(
        data.name.trim().toLowerCase()
      );
      if (id !== -1) {
        this.datasourceFieldsNames.splice(id, 1);
        this.datasourceFieldsNames = [...this.datasourceFieldsNames];
      }
    }
    if (this.dataSourcesFields[this.edgeId]) {
      let idx = this.dataSourcesFields[this.edgeId].indexOf(this.editedTopic);
      if (idx >= 0) this.dataSourcesFields[this.edgeId].splice(idx, 1);
      this.dataSourcesFields[this.edgeId] = [
        ...this.dataSourcesFields[this.edgeId],
      ];
    }

    if (this.tempEditObject[data.key]) delete this.tempEditObject[data.key];
    this.editRow = null;
    const idx = this.fields.findIndex(v => v.key === data.key);
    if (idx !== -1) {
      this.fields.splice(idx, 1);
      this.fields = [...this.fields];
    }
  }

  clickFieldRow(data) {
    if (this.editRow !== data.key) {
      this.edit(data);
    }
  }
  checkDuplicates(entity, value) {
    if (entity === 'dsName') {
      if (this.datasourceNames.includes(value.trim().toLowerCase()))
        this.duplicateDatasourceNameFound = true;
      else this.duplicateDatasourceNameFound = false;
    }
    if (entity === 'topicName') {
      if (this.datasourceFieldsNames.includes(value.trim().toLowerCase()))
        this.duplicateFieldNameFound = true;
      else this.duplicateFieldNameFound = false;
    }

    if (entity === 'topic') {
      if (this.sensorAuth === 'PASSWORD') {
        let regex = /^[^:@]*$/;
        if (!value.match(regex)) this.invalidTopic = true;
        else {
          this.invalidTopic = false;
          value = this.populatedUrl + value;
        }
      }

      if (
        this.dataSourcesFields[this.edgeId] &&
        this.dataSourcesFields[this.edgeId].includes(
          value.trim().toLowerCase()
        ) &&
        value.trim().toLowerCase() !== this.editedTopic.trim().toLowerCase()
      )
        this.duplicateTopicNameFound = true;
      else this.duplicateTopicNameFound = false;
    }
  }
  validateIP(entity, idx) {
    let regex = /^(\d|[1-9]\d|1\d\d|2([0-4]\d|5[0-5]))$/;
    if (entity.num.length !== 0 && entity.num.includes('.')) {
      let entities = entity.num.split('.');
      if (entities.length === 4) {
        entities.forEach(e => {
          if (e.match(regex)) {
          }
        });
        if (idx === 'ip') {
          this.dsIPAddress = '';
          this.dsIP1.num = entities[0];
          this.dsIP2.num = entities[1];
          this.dsIP3.num = entities[2];
          this.dsIP4.num = entities[3];
          entities.forEach(e => {
            this.dsIPAddress += e + '.';
          });

          this.dsIPAddress = this.dsIPAddress.substr(
            0,
            this.dsIPAddress.length - 1
          );
        }
      } else {
        entity.error = true;
        this.invalidInput = true;
      }
      return;
    }
    if (entity.num.length !== 0 && !entity.num.match(regex)) {
      entity.error = true;
      this.invalidInput = true;
      return;
    }
    if (idx === 'ip') {
      this.dsIPAddress =
        this.dsIP1.num +
        '.' +
        this.dsIP2.num +
        '.' +
        this.dsIP3.num +
        '.' +
        this.dsIP4.num;
    }
    entity.error = false;
    this.invalidInput = false;
  }
}
