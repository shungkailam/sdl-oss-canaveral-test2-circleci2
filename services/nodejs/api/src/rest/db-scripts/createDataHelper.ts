// script to help create tenant mock data
// The script overrides the template data with the user specified overrides.
// It can also substitute some keys like {{edge}}, {{idx}}

import {
  Tenant,
  User,
  Edge,
  Category,
  Script,
  DataSource,
  DataSourceFieldInfo,
  DataSourceFieldSelector,
  DataStream,
  CategoryInfo,
  CloudCreds,
  BaseModel,
  DataStreamDestination,
  EdgeStreamType,
  CloudType,
  UserRoleType,
  Project,
  DockerProfile,
  ScriptRuntime,
  Application,
  ApplicationStatus,
  EdgeInfo,
} from '../model/index';
import { DocType } from '../model/baseModel';
import { getSha256 } from '../util/cryptoUtil';
import * as uuidv4 from 'uuid/v4';
import platformService from '../services/platform.service';
import { initSequelize } from '../sql-api/baseApi';
import { isSQL } from '../db-configurator/dbConfigurator';
import { TenantBuilder } from './tenantBuilder';
import { searchObjects } from '../es-api/baseApi';
import { Sequelize } from 'sequelize-typescript';
import { deleteTenant } from './common';
import { APP_STATUS } from './dataDB';

const userTemplate: User = {
  tenantId: '',
  id: '',
  email: '',
  name: '',
  password: '',
  role: UserRoleType.INFRA_ADMIN,
};

const edgeTemplate: Edge = {
  tenantId: '',
  id: '',
  name: 'Cupertino Store',
  description: ' ',
  connected: false,
  serialNumber: '',
  ipAddress: '10.1.2.37',
  gateway: '10.1.2.1',
  subnet: '10.1.0.0',
  edgeDevices: 1,
  storageCapacity: 100,
  storageUsage: 5,
};

const categoryTemplate: Category = {
  tenantId: '',
  id: '',
  name: 'Counter Number',
  purpose: '',
  values: ['1', '2', '3'],
};

const dataSourceTemplate: DataSource = {
  tenantId: '',
  edgeId: '',
  id: uuidv4(),
  name: 'POS Camera 1',
  type: 'Sensor',
  sensorModel: '',
  connection: 'Secure',
  fields: [
    {
      name: 'Camera Feed',
      fieldType: 'Image',
      mqttTopic: 'rtsp://camadmin:campwd@10.15.232.#/Streaming/Channels/102',
    },
  ],
  selectors: [
    {
      id: '',
      value: '1',
      scope: ['__ALL__'],
    },
  ],
  protocol: 'RTSP',
  authType: 'PASSWORD',
};

const dataStreamTemplate: DataStream = {
  tenantId: '',
  id: '',
  name: 'Cash_RegisterCam',
  dataType: 'Image',
  origin: 'Data Source',
  originSelectors: [
    {
      id: '',
      value: '1',
    },
  ],
  destination: DataStreamDestination.Cloud,
  edgeStreamType: EdgeStreamType.ElasticSearch,
  size: 0,
  enableSampling: true,
  samplingInterval: 60,
  transformationArgsList: [],
  dataRetention: [],
  projectId: null,
  endPoint: null,
};

const cloudProfileTemplate: CloudCreds = {
  tenantId: '',
  id: '',
  type: CloudType.AWS,
  name: 'AWS Profile',
  description: 'Main AWS account profile',
  awsCredential: JSON.stringify({
    accessKey: 'access_key_here',
    secret: 'secret_here',
  }),
};

const scriptTemplate: Script = {
  tenantId: '',
  id: '',
  name: 'People counter',
  type: 'Transformation',
  language: 'python',
  environment: 'tensorflow-python',
  code: `def main: pass`,
  params: [],
  projectId: null,
  builtin: false,
  runtimeId: null,
  runtimeTag: '',
};

const projectTemplate: Project = {
  tenantId: '',
  id: '',
  name: 'Default Project',
  description: 'default project',
  cloudCredentialIds: [],
  dockerProfileIds: [],
  users: [],
  edgeSelectorType: 'Explicit',
  edgeIds: [],
  edgeSelectors: [],
};

const dockerProfileTemplate: DockerProfile = {
  tenantId: '',
  id: '',
  name: 'flask-web-hub',
  description: 'flaskwebhub',
  type: 'ContainerRegistry',
  server: 'https://index.docker.io/v1/',
  userName: 'demo',
  email: 'demo@nutanix.com',
  pwd: 'Iphone@123',
  cloudCredsID: null,
  credentials: '',
};
const applicationTemplate: Application = {
  tenantId: '',
  id: '',
  name: 'docker-flaskweb-hub',
  description: 'docker hub3docker hub3',
  // originSelectors: [],
  projectId: null,
  yamlData:
    'apiVersion: v1\nkind: Pod\nmetadata:\n  name: flask-web-server-waldot-repo\n  labels:\n    app: flask-web-server-waldot-repo\nspec:\n  containers:\n  - image: ntayal/myrepo:latest\n    name: flask-web-server-waldot-repo\n    imagePullPolicy: Always\n    ports:\n    - containerPort: 5000\n---\nkind: Service\napiVersion: v1\nmetadata:\n  name: flask-web-server-svc-waldot-repo\nspec:\n  selector:\n    app: flask-web-server-waldot-repo\n  ports:\n  - protocol: TCP\n    name: flask-web-server-waldot-repo\n    port: 5000\n    targetPort: 5000',
};

const scriptRuntimeTemplate: ScriptRuntime = {
  tenantId: '',
  id: '',
  name: 'My Node Env',
  description: 'My NodeJS Runtime',
  language: 'node',
  builtin: false,
  dockerRepoURI: 'node-env-2',
  dockerProfileID: '',
  dockerfile: '',
  projectId: null,
};
const applicationStatusTemplate: ApplicationStatus = {
  ...APP_STATUS,
  tenantId: '',
  edgeId: '',
  applicationId: '',
};

const edgeInfoTemplate: EdgeInfo = {
  tenantId: '',
  edgeId: '',
  id: '',
  NumCPU: '',
  TotalMemoryKB: '',
  TotalStorageKB: '',
  GPUInfo: '',
  CPUUsage: '',
  MemoryFreeKB: '',
  StorageFreeKB: '',
  GPUUsage: '',
};

export function getVersion() {
  return Math.floor(Date.now() / 1000);
}

// Recursively substitute params {{param}} in the object and
// create new objects on substitution.
function substitute(obj: any, properties: any): any {
  const type = typeof obj;
  if (type == 'string') {
    let result = obj;
    Object.keys(properties).forEach(key => {
      result = result.replace(`{{${key}}}`, properties[key]);
    });
    return result;
  }
  if (type == 'object') {
    if (Array.isArray(obj)) {
      const newArray: any[] = [];
      for (var key in obj) {
        newArray.push(substitute(obj[key], properties));
      }
      return newArray;
    } else {
      // Regular object
      const newObj: any = {};
      Object.keys(obj).forEach(key => {
        newObj[key] = substitute(obj[key], properties);
      });
      return newObj;
    }
  }
  return obj;
}

function updateEdgeIP(edge: Edge, index: number) {
  if (index) {
    const ds = edge.ipAddress.split('.').map(x => parseInt(x, 10));
    let [d0, d1, d2, d3] = ds;
    d3 += index;
    if (d3 > 255) {
      const dd = Math.floor(d3 / 256);
      d3 -= dd * 256;
      d2 += dd;
    }
    edge.ipAddress = [d0, d1, d2, d3].join('.');
  }
}

export class CreateDataHelper {
  tenantBuilder: TenantBuilder = null;

  constructor(private tenant: Tenant, private sql: Sequelize) {
    this.tenantBuilder = new TenantBuilder(tenant, sql);
  }

  // Delete all the records belonging to this email/tenant
  async cleanDB() {
    return deleteTenant(this.sql, this.tenant.id);
  }

  addUser(name: string, email: string, password: string): User {
    const version = getVersion();
    const user: User = Object.assign({}, userTemplate, {
      tenantId: this.tenant.id,
      id: uuidv4(),
      name,
      email,
      password: getSha256(password),
      version,
    });
    console.log('>>> adding user:', user);
    this.tenantBuilder.addUser(user);
    return user;
  }

  addEdges(names: string[]): Edge[] {
    return names.map((name, i) => {
      const version = getVersion();
      const edge: Edge = Object.assign({}, edgeTemplate, {
        version,
        name,
        tenantId: this.tenant.id,
        id: `eid-demo-${uuidv4().substring(9)}`,
        serialNumber: uuidv4(),
      });
      updateEdgeIP(edge, i);
      console.log('>>> adding edge:', edge);
      this.tenantBuilder.addEdge(edge);
      return edge;
    });
  }

  addCategories(properties: any): Category[] {
    const categories: Category[] = [];
    Object.keys(properties).forEach(key => {
      const version = getVersion();
      const category: Category = Object.assign({}, categoryTemplate, {
        version,
        tenantId: this.tenant.id,
        id: uuidv4(),
        name: key,
        values: properties[key],
      });
      this.tenantBuilder.addCategory(category);
      console.log('>>> adding category:', category);
      categories.push(category);
    });
    return categories;
  }

  // Substitution keys are {{ip}} edge loop, {{field}} for field index, {{edge}} for edge name
  // and {{selectorValue}} for value
  addDataSources(
    selectorValues: string[][],
    edges: Edge[],
    dataSourceOverride: any,
    fieldCountCallback
  ): DataSource[] {
    var selectorsIdx = 0;
    const dataSources: DataSource[] = [];
    const substitutes = {};
    edges.forEach(edge => {
      const ip = selectorsIdx + 10;
      const version = getVersion();
      let dataSource: DataSource = Object.assign(
        {},
        dataSourceTemplate,
        dataSourceOverride,
        {
          version,
          tenantId: this.tenant.id,
          id: uuidv4(),
          edgeId: edge.id,
        }
      );
      dataSource = substitute(dataSource, {
        edge: edge.name,
        ip: ip.toString(),
      });
      const dataSourceInfos: DataSourceFieldInfo[] = [];
      for (
        var fieldIdx = 0;
        fieldIdx < fieldCountCallback(edge.name);
        fieldIdx++
      ) {
        // Pick the first field
        const dataSourceInfo: DataSourceFieldInfo = substitute(
          dataSource.fields[0],
          {
            edge: edge.name,
            field: fieldIdx.toString(),
            ip: ip.toString(),
          }
        );
        dataSourceInfos.push(dataSourceInfo);
      }
      dataSource.fields = dataSourceInfos;
      const selectors: DataSourceFieldSelector[] = [];
      const values = selectorValues[selectorsIdx++];
      var valueIdx = 0;
      dataSource.selectors.forEach(s => {
        console.log(values);
        const selector: DataSourceFieldSelector = substitute(s, {
          edge: edge.name,
          ip: ip.toString(),
          selectorValue: values[valueIdx++],
        });
        selectors.push(selector);
      });
      dataSource.selectors = selectors;
      console.log('>>> adding datasource:', dataSource);
      this.tenantBuilder.addDataSource(dataSource);
      dataSources.push(dataSource);
    });
    return dataSources;
  }

  addDataStream(
    originSelectorValues: string[][],
    dataStreamsOverride: any
  ): DataStream {
    const version = getVersion();
    const dataStream: DataStream = Object.assign(
      {},
      dataStreamTemplate,
      dataStreamsOverride,
      {
        version,
        tenantId: this.tenant.id,
        id: uuidv4(),
      }
    );
    const originalSelectors: CategoryInfo[] = [];
    originSelectorValues.forEach(values => {
      var valueIdx = 0;
      dataStream.originSelectors.forEach(originalSelector => {
        const categoryInfo: CategoryInfo = substitute(originalSelector, {
          selectorValue: values[valueIdx++],
        });
        originalSelectors.push(categoryInfo);
      });
    });
    dataStream.originSelectors = originalSelectors;
    console.log('>>> adding datastream:', dataStream);
    this.tenantBuilder.addDataStream(dataStream);
    return dataStream;
  }

  addCloudProfile(cloudProfileOverride: any): CloudCreds {
    const version = getVersion();
    const cloudCreds: CloudCreds = Object.assign(
      {},
      cloudProfileTemplate,
      cloudProfileOverride,
      {
        version,
        tenantId: this.tenant.id,
        id: uuidv4(),
      }
    );
    console.log('>>> adding cloud profile:', cloudCreds);
    this.tenantBuilder.addCloudProfile(cloudCreds);
    return cloudCreds;
  }

  addScript(scriptOverride: any): Script {
    const version = getVersion();
    const script: Script = Object.assign({}, scriptTemplate, scriptOverride, {
      version,
      tenantId: this.tenant.id,
      id: uuidv4(),
    });
    console.log('>>> adding script:', script);
    this.tenantBuilder.addScript(script);
    return script;
  }

  addDockerProfile(dockerProfileOverride: any): DockerProfile {
    const version = getVersion();
    const dockerProfile: DockerProfile = Object.assign(
      {},
      dockerProfileTemplate,
      dockerProfileOverride,
      {
        version,
        tenantId: this.tenant.id,
        id: uuidv4(),
      }
    );
    console.log('>>> adding docker profile:', dockerProfile);
    this.tenantBuilder.addDockerProfile(dockerProfile);
    return dockerProfile;
  }

  addProject(projectOverride: any): Project {
    const version = getVersion();
    const project: Project = Object.assign(
      {},
      projectTemplate,
      projectOverride,
      {
        version,
        tenantId: this.tenant.id,
        id: uuidv4(),
      }
    );
    console.log('>>> adding project:', project);
    this.tenantBuilder.addProject(project);
    return project;
  }

  addApplication(applicationOverride: any): Application {
    const version = getVersion();
    const application: Application = Object.assign(
      {},
      applicationTemplate,
      applicationOverride,
      {
        version,
        tenantId: this.tenant.id,
        id: uuidv4(),
      }
    );
    console.log('>>> adding application:', application);
    this.tenantBuilder.addApplication(application);
    return application;
  }

  addScriptRuntime(scriptRuntimeOverride: any): ScriptRuntime {
    const version = getVersion();
    const scriptRuntime: ScriptRuntime = Object.assign(
      {},
      scriptRuntimeTemplate,
      scriptRuntimeOverride,
      {
        version,
        tenantId: this.tenant.id,
        id: uuidv4(),
      }
    );
    console.log('>>> adding script runtime:', scriptRuntime);
    this.tenantBuilder.addScriptRuntime(scriptRuntime);
    return scriptRuntime;
  }

  addApplicationStatus(applicationStatusOverride: any): ApplicationStatus {
    const version = getVersion();
    const applicationStatus: ApplicationStatus = Object.assign(
      {},
      applicationStatusTemplate,
      applicationStatusOverride,
      {
        version,
        tenantId: this.tenant.id,
      }
    );
    console.log('>>> adding application status:', applicationStatus);
    this.tenantBuilder.addApplicationStatus(applicationStatus);
    return applicationStatus;
  }
  addEdgeInfo(edgeInfoOverride: any): EdgeInfo {
    const version = getVersion();
    const edgeInfo: EdgeInfo = Object.assign(
      {},
      edgeInfoTemplate,
      edgeInfoOverride,
      {
        version,
        tenantId: this.tenant.id,
      }
    );
    console.log('>>> adding edgeInfo:', edgeInfo);
    this.tenantBuilder.addEdgeInfo(edgeInfo);
    return edgeInfo;
  }

  async create() {
    return new Promise(async (resolve, reject) => {
      try {
        await this.tenantBuilder.create();
        resolve();
      } catch (err) {
        console.log(err);
        reject();
      }
    });
  }
}
