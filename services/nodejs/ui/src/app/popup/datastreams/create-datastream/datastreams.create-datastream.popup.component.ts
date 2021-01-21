import { Component } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import * as omit from 'object.omit';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../../../services/registry.service';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import {
  DataStream,
  DATA_TYPES,
  CategoryInfo,
  AWS_REGION,
  AWS_REGIONS,
  GCP_REGION,
  GCP_REGIONS,
  AWSStreamType,
  AWSStreamTypes,
  GCPStreamType,
  GCPStreamTypes,
  EdgeStreamType,
  EdgeStreamTypes,
  DataStreamDestination,
  CloudType,
  TransformationArgs,
  ScriptParamValue,
} from '../../../model/index';
import {
  datasourceMatchOriginSelectors,
  getDatasourceMatchingSensorsCount,
} from '../../../utils/modelUtil';
import * as uuidv4 from 'uuid/v4';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

// must match dataStream.ts
const ORIGIN_DATA_STREAM = 'Data Stream';
const ORIGIN_DATA_SOURCE = 'Data Source';

function makeTransformationInfo(): TransformationArgs {
  return {
    transformationId: '',
    args: [],
  };
}
@Component({
  selector: 'app-datastreams-create-datastream-popup',
  templateUrl: './datastreams.create-datastream.popup.component.html',
  styleUrls: ['./datastreams.create-datastream.popup.component.css'],
})
export class DataStreamsCreateDataStreamPopupComponent extends TableBaseComponent {
  dataTypes = DATA_TYPES;
  streamName = '';
  dataType = '';
  origin = '';

  destination = '';
  streamType = 'Pub Sub';
  edgeStreamType = EdgeStreamType.None;
  edgeStreamTypes = EdgeStreamTypes;
  enableSampling = false;
  samplingInterval = 1;
  samplingIntervalUnits = 'second';
  transformationInfoList: TransformationArgs[] = [makeTransformationInfo()];

  cloudType = CloudType.AWS;
  cloudCredsId = null;
  awsStreamType = AWSStreamType.Kinesis;
  awsStreamTypes = AWSStreamTypes;
  gcpStreamType = GCPStreamType.PubSub;
  gcpStreamTypes = GCPStreamTypes;
  awsRegion: AWS_REGION = AWS_REGION.US_WEST_2;
  awsRegions = AWS_REGIONS;
  gcpRegion: GCP_REGION = GCP_REGION.US_WEST1;
  gcpRegions = GCP_REGIONS;
  cloudCredsList = [];
  allCloudCredsList = [];
  showTrans = false;
  showDes = false;
  duplicateStream = false;
  samplingValueError = false;

  dataStreams = [];
  scripts = [];
  categories = [];
  projects = [];
  catInfos = [
    {
      id: '',
      value: '',
      values: [],
    },
  ];
  projectId = '';
  projectName = '';
  isConfirmLoading = false;
  dataStreamId = '';

  queryParamSub = null;
  dataStream = null;

  // subscribe to router event for upload script
  routerEventUrl = '/datastreams/list(popup:datastreams/create-datastream)';
  objectRecognitionId = null;

  getCountQuery = {};
  dataSourceCount = 0;
  affectedEdges = [];
  sensorCount = 0;

  datasources = [];
  edges = [];
  context = '';
  allDatastreams = [];

  constructor(
    router: Router,
    private route: ActivatedRoute,
    private http: Http,
    private regService: RegistryService
  ) {
    super(router);
    this.fetchCategories().then(
      () => {
        this.queryParamSub = this.route.queryParams.subscribe(params => {
          if (params && params.projectId) {
            this.projectId = params.projectId;
            this.context = 'project';
          } else if (params && params.id) {
            // id param exists - update case
            this.showTrans = true;
            const ds: DataStream = this.regService.get(params.id);
            if (ds) {
              this.streamName = ds.name;
              this.dataStream = ds;
              this.dataType = ds.dataType;
              this.origin = ds.origin;
              this.destination = ds.destination;
              this.cloudType = ds.cloudType;
              this.edgeStreamType = ds.edgeStreamType;
              this.awsStreamType = ds.awsStreamType;
              this.gcpStreamType = ds.gcpStreamType;
              this.awsRegion = ds.awsCloudRegion;
              this.gcpRegion = ds.gcpCloudRegion;
              this.cloudCredsId = ds.cloudCredsId;
              this.projectId = ds.projectId;
              this.enableSampling = ds.enableSampling;
              this.samplingInterval = ds.samplingInterval;
              this.samplingIntervalUnits = 'millisecond';
              let sampleVal = this.samplingInterval;
              if (sampleVal % 1000 === 0) {
                sampleVal = sampleVal / 1000;
                this.samplingIntervalUnits = 'second';
                if (sampleVal % 60 === 0) {
                  sampleVal = sampleVal / 60;
                  this.samplingIntervalUnits = 'minute';
                  if (sampleVal % 60 === 0) {
                    sampleVal = sampleVal / 60;
                    this.samplingIntervalUnits = 'hour';
                    if (sampleVal % 24 === 0) {
                      sampleVal = sampleVal / 24;
                      this.samplingIntervalUnits = 'day';
                    }
                  }
                }
                this.samplingInterval = sampleVal;
              }
              if (ds.transformationArgsList.length) {
                this.transformationInfoList = ds.transformationArgsList.map(
                  t => ({
                    transformationId: t.transformationId,
                    args: t.args.slice(),
                  })
                );
              }
              if (ds.origin === ORIGIN_DATA_STREAM) {
                this.dataStreamId = ds.originId;
              }
              const cis = [];
              if (ds.originSelectors && ds.originSelectors.length) {
                ds.originSelectors.forEach(os => {
                  if (os.id) {
                    const cat = this.categories.find(c => c.id === os.id);
                    if (cat) {
                      cis.push({
                        id: os.id,
                        value: os.value,
                        values: cat.values,
                      });
                    } else {
                      console.error(
                        '>>> failed to find category with id: ' + os.id,
                        os
                      );
                    }
                  }
                });
                if (cis.length) {
                  this.catInfos = cis;
                  this.onUpdateCatInfo();
                }
              }
            }
          }
          this.fetchDataStreams();
          this.fetchCloudCreds();
          this.fetchScripts();
        });
      },
      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.fetchCategories()
        );
      }
    );

    this.fetchProjects();
    this.fetchDataStreamsByUser();
  }

  onClickDataSource(target) {
    this.origin = target.currentTarget.innerText.trim();
  }

  onClickAddModules(target, ele) {
    this.showTrans = true;
    //ele.transformationId = ele.id;
    if (ele && ele.id) {
      this.transformationInfoList[0] = {
        args: this.getTransformationArgs(ele.id),
        transformationId: ele.id,
      };
    }
  }

  fetchDataStreamsByUser() {
    this.http
      .get('/v1/datastreams', getHttpRequestOptions())
      .toPromise()
      .then(
        res => {
          this.allDatastreams = res.json();
        },

        rej => {
          handleAuthError(null, rej, this.router, this.http, () =>
            this.fetchDataStreamsByUser()
          );
        }
      );
  }

  onClickAddDestination(target) {
    this.showDes = true;
    this.destination = target.currentTarget.innerText.trim();
    this.getRightArrowHeight();
    this.getRightArrowPos();
  }

  fetchScripts() {
    this.http
      .get(`/v1/projects/${this.projectId}/scripts`, getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          this.scripts = x.json().filter(s => s.type === 'Transformation');
          const ors = this.scripts.find(s => s.name === 'Object Recognition');
          if (ors) {
            this.objectRecognitionId = ors.id;
          } else {
            this.objectRecognitionId = null;
          }
        },
        rej => {
          handleAuthError(
            null,
            rej,
            this.router,
            this.http,
            () => this.fetchScripts()
            // function() { this.getApplications(); }
          );
        }
      );
  }

  fetchProjects() {
    this.http
      .get('/v1/projects', getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          this.projects = x.json();
          const pro = this.projects.find(p => p.id === this.projectId);
          if (pro) {
            this.projectName = pro.name;
          }
        },
        rej => {
          handleAuthError(
            null,
            rej,
            this.router,
            this.http,
            () => this.fetchProjects()
            // function() { this.getApplications(); }
          );
        }
      );
  }

  fetchCategories() {
    return this.http
      .get('/v1/categories', getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          this.categories = x.json();
        },
        rej => {
          handleAuthError(
            null,
            rej,
            this.router,
            this.http,
            () => this.fetchCategories()
            // function() { this.getApplications(); }
          );
        }
      );
  }

  fetchDataStreams() {
    return this.http
      .get(
        `/v1/projects/${this.projectId}/datastreams`,
        getHttpRequestOptions()
      )
      .toPromise()
      .then(
        x => {
          if (this.dataStream) {
            // don't allow stream as its own origin
            this.dataStreams = x
              .json()
              .filter(ds => ds.id !== this.dataStream.id);
          } else {
            this.dataStreams = x.json();
          }
        },
        rej => {
          handleAuthError(
            null,
            rej,
            this.router,
            this.http,
            () => this.fetchDataStreams()
            // function() { this.getApplications(); }
          );
        }
      );
  }

  disableCategory(ci, cv) {
    const matchedCat = this.catInfos.filter(c => c.id === ci.id);
    const found = matchedCat.find(mc => mc.value === cv);
    if (found) {
      return true;
    }
    return false;
  }

  fetchCloudCreds() {
    return this.http
      .get(`/v1/projects/${this.projectId}/cloudcreds`, getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          this.allCloudCredsList = x.json();
          this.changeCloudType(false);
        },
        rej => {
          handleAuthError(
            null,
            rej,
            this.router,
            this.http,
            () => this.fetchCloudCreds()
            // function() { this.getApplications(); }
          );
        }
      );
  }

  changeCloudType(resetInitialVal) {
    this.cloudCredsList = this.allCloudCredsList.filter(
      e => e.type === this.cloudType
    );
    if (resetInitialVal || !this.dataStream) {
      this.cloudCredsId =
        this.cloudCredsList.length > 0 ? this.cloudCredsList[0].id : null;
    }
  }

  isCreateDisabled() {
    return (
      (this.enableSampling && this.samplingValueError) ||
      this.duplicateStream ||
      this.projectId === '' ||
      this.streamName === '' ||
      this.destination === '' ||
      (this.destination === 'Cloud' && !this.cloudCredsId) ||
      this.isTransformationInfoIncomplete() ||
      (this.origin === ORIGIN_DATA_SOURCE && this.isCatInfoIncomplete()) ||
      (this.origin === ORIGIN_DATA_STREAM && this.dataStreamId === '') ||
      this.origin === '' ||
      !this.showTrans
    );
  }

  isCatInfoIncomplete() {
    return this.catInfos.some(ci => ci.id === '' || ci.value === '');
  }

  isTransformationInfoIncomplete() {
    if (this.transformationInfoList.length > 1) {
      // all must have transformationId in this case
      return this.transformationInfoList.some(x => !x.transformationId);
    }
    return false;
  }

  onCreateDataStream() {
    const tenantId = this.regService.get(REG_KEY_TENANT_ID);
    const name = this.streamName;
    const dataType = this.dataType;
    const origin: 'Data Source' | 'Data Stream' =
      this.origin === ORIGIN_DATA_SOURCE
        ? ORIGIN_DATA_SOURCE
        : ORIGIN_DATA_STREAM;
    const originSelectors = this.catInfos.map(ci => ({
      id: ci.id,
      value: ci.value,
    }));
    const destination: DataStreamDestination =
      this.destination === 'Edge'
        ? DataStreamDestination.Edge
        : DataStreamDestination.Cloud;
    // const streamType = this.streamType;
    const cloudType = this.cloudType;
    const cloudCredsId = this.cloudCredsId;
    const edgeStreamType = this.edgeStreamType;
    const awsStreamType = this.awsStreamType;
    const gcpStreamType = this.gcpStreamType;
    const awsCloudRegion = this.awsRegion;
    const gcpCloudRegion = this.gcpRegion;
    const size = 0;
    switch (this.samplingIntervalUnits) {
      case 'second':
        this.samplingInterval *= 1000;
        break;
      case 'minute':
        this.samplingInterval *= 60000;
        break;
      case 'hour':
        this.samplingInterval *= 3600000;
        break;
      case 'day':
        this.samplingInterval *= 86400000;
        break;
    }
    this.samplingIntervalUnits = 'millisecond';
    const enableSampling = this.enableSampling;
    const samplingInterval = this.samplingInterval;
    const transformationArgsList = this.transformationInfoList
      // .map(({transformationId, args}) => ({
      //   transformationId,
      //   args,
      // }))
      .filter(t => Boolean(t.transformationId)); // filter for len == 1 and empty transformationId case
    const retType: 'Size' | 'Time' = 'Size';
    // TODO FIXME - hardcode to 200 TB as an example - comment out b/c cause ES exception
    const dataRetention = [
      // {
      //   type: retType,
      //   limit: 200000,
      // },
    ];
    const id = uuidv4();
    const projectId = this.projectId;

    const datastream: DataStream = {
      id,
      tenantId,
      name,
      dataType,
      origin,
      originSelectors,
      destination,
      cloudType,
      cloudCredsId,
      edgeStreamType,
      awsStreamType,
      gcpStreamType,
      awsCloudRegion,
      gcpCloudRegion,
      size,
      enableSampling,
      samplingInterval,
      transformationArgsList,
      dataRetention,
      projectId,
    };
    if (origin === ORIGIN_DATA_STREAM) {
      datastream.originId = this.dataStreamId;
      datastream.dataType = 'Custom';
    }
    let method = 'post';
    if (this.dataStream) {
      datastream.id = this.dataStream.id;
      method = 'put';
    }

    this.isConfirmLoading = true;
    this.http[method]('/v1/datastreams', datastream, getHttpRequestOptions())
      .toPromise()
      .then(
        x => {
          this.isConfirmLoading = false;
          this.onClosePopup();
        },
        e => {
          const warning =
            'Failed to ' +
            (method === 'post' ? 'create' : 'update') +
            ' datastreams';
          this.isConfirmLoading = false;
          handleAuthError(() => alert(warning), e, this.router, this.http, () =>
            this.onCreateDataStream()
          );
          this.onClosePopup();
        }
      );
  }

  onChangeSelectCategory(ci) {
    const cat = this.categories.find(c => c.id === ci.id);
    ci.values = cat.values;
    ci.value = '';
    this.onUpdateCatInfo();
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
    this.onUpdateCatInfo();
  }

  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onClickUploadScript() {
    this.router.navigate([{ outlets: { popup2: ['scripts', 'upload'] } }]);
  }

  onClickCloseCircle(index) {
    this.transformationInfoList.splice(index, 1);
  }

  onClickPlusCircle(index) {
    this.transformationInfoList.splice(index + 1, 0, makeTransformationInfo());
  }

  getRightArrowPosNumber() {
    const tl = this.transformationInfoList;
    return tl.slice(0, tl.length - 1).reduce((prev, curr, index) => {
      const i0 = index === 0;
      const extra =
        curr.transformationId === this.objectRecognitionId
          ? 166
          : curr.args.length * 83;
      return prev + (i0 && this.enableSampling ? 191 : i0 ? 149 : 144) + extra;
    }, 153);
  }

  getRightArrowPos() {
    const p = this.getRightArrowPosNumber();
    return `${p}px`;
  }
  getRightArrowHeight() {
    const p = this.getRightArrowPosNumber() - 108;
    return `${p}px`;
  }

  onUpdateCatInfo() {
    const tempGetCountQuery = {};
    const categoryInfoList: CategoryInfo[] = this.catInfos
      .filter(cat => cat.value)
      .map(cat => ({
        id: cat.id,
        value: cat.value,
      }));

    if (categoryInfoList.length === 0) {
      this.dataSourceCount = 0;
    } else {
      tempGetCountQuery['nested'] = categoryInfoList;
      tempGetCountQuery['fieldType'] = this.dataType;

      this.getCountQuery = tempGetCountQuery;
      // this.fetchDataSourceCount();
      this.updateAffectedEdges(categoryInfoList);
    }
  }

  onChangeDataType() {
    this.onUpdateCatInfo();
  }

  async updateAffectedEdges(catInfoList: CategoryInfo[]) {
    let promise = [];
    promise.push(
      this.http
        .get(
          `/v1/projects/${this.projectId}/datasources`,
          getHttpRequestOptions()
        )
        .toPromise()
    );
    promise.push(
      this.http
        .get(`/v1/projects/${this.projectId}/edges`, getHttpRequestOptions())
        .toPromise()
    );
    Promise.all(promise).then(
      res => {
        if (res.length === 2) {
          this.datasources = res[0].json();
          this.edges = res[1].json();
          this.sensorCount = 0;
          const dss = this.datasources.filter(ds =>
            datasourceMatchOriginSelectors(ds, catInfoList, this.dataType)
          );
          if (dss.length) {
            dss.forEach(
              ds =>
                (ds.sensorCount = getDatasourceMatchingSensorsCount(
                  ds,
                  catInfoList,
                  this.dataType
                ))
            );
            this.edges.forEach(edge => {
              edge.dsCount = 0;
              edge.sensorCount = 0;
            });
            dss.forEach(ds => {
              const edge = this.edges.find(e => e.id === ds.edgeId);
              if (edge) {
                edge.dsCount++;
                edge.sensorCount += ds.sensorCount;
              }
              this.sensorCount = this.sensorCount + ds.sensorCount;
            });
            this.affectedEdges = this.edges.filter(edge => edge.dsCount > 0);
            // also calculate dataSourceCount
            this.dataSourceCount = dss.length;
          } else {
            this.affectedEdges = [];
            this.dataSourceCount = 0;
          }
        }
      },

      rej => {
        handleAuthError(null, rej, this.router, this.http, () =>
          this.updateAffectedEdges(catInfoList)
        );
      }
    );
  }

  onTransformationIdChange(ti: TransformationArgs) {
    if (ti.transformationId) {
      ti.args = this.getTransformationArgs(ti.transformationId);
    } else {
      ti.args = [];
    }
  }
  getTransformationArgs(transformationId: string): ScriptParamValue[] {
    if (this.scripts) {
      const script = this.scripts.find(s => s.id === transformationId);
      if (script && script.params && script.params.length) {
        return script.params.map(({ name, type }) => ({
          name,
          type,
          value: '',
        }));
      }
    }
    return [];
  }

  checkNameDuplicate(stream) {
    const s = stream.trim().toLowerCase();
    this.duplicateStream = false;
    this.allDatastreams.forEach(d => {
      const n = d.name.trim().toLowerCase();
      if (n === s) {
        this.duplicateStream = true;
        return;
      }
    });
  }

  dataSteamNameCheck(target) {
    const regex = RegExp(/[a-z0-9\-]/);
    if (!regex.test(target.key)) {
      return false;
    }
  }

  samplingIntervalChange() {
    if (this.samplingInterval <= 0) {
      this.samplingValueError = true;
    } else {
      this.samplingValueError = false;
    }
  }
}
