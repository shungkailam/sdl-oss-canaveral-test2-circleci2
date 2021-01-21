// script to help create tenant mock data

import {
  Tenant,
  User,
  Edge,
  Category,
  Script,
  DataSource,
  DataStream,
  CloudCreds,
  BaseModel,
  DataStreamDestination,
  EdgeStreamType,
  CloudType,
  Application,
  DockerProfile,
  ScriptRuntime,
  ApplicationStatus,
  UserRoleType,
  Project,
  EdgeInfo,
} from '../model/index';
import { DocType } from '../model/baseModel';
import { getSha256 } from '../util/cryptoUtil';
import * as uuidv4 from 'uuid/v4';
import platformService from '../services/platform.service';
import { initSequelize } from '../sql-api/baseApi';
import { isSQL } from '../db-configurator/dbConfigurator';
import { TenantBuilder } from './tenantBuilder';
import { Sequelize } from 'sequelize-typescript';
import { getVersion } from './createDataHelper';
import { APP_STATUS } from './dataDB';
import { deleteTenant } from './common';
import { getDefaultProjectId } from '../../scripts/common';

//
// An example script to show how to use TenantBuilder
// to populate DB objects for a new Tenant
//
async function main() {
  let sql: Sequelize = initSequelize();
  try {
    const version = getVersion();
    const tenantId = 'tenant-id-smart-retail';
    const tenant: Tenant = {
      version,
      id: tenantId,
      name: 'Smart Retail',
      token: await platformService.getKeyService().genTenantToken(),
    };

    await deleteTenant(sql, tenantId);

    const tenantBuilder = new TenantBuilder(tenant, sql);

    const catLocation: Category = {
      tenantId,
      version,
      id: uuidv4(),
      name: 'Counter Number',
      purpose: '',
      values: ['1', '2', '3'],
    };

    const cloudProfile: CloudCreds = {
      tenantId,
      version,
      id: uuidv4(),
      type: CloudType.AWS,
      name: 'AWS Profile',
      description: 'Main AWS account profile',
      awsCredential: JSON.stringify({
        accessKey: 'access_key_here',
        secret: 'secret_here',
      }),
    };

    const dockerProfile: DockerProfile = {
      tenantId,
      version,
      id: uuidv4(),
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

    const edge: Edge = {
      tenantId,
      version,
      id: uuidv4(),
      name: 'Cupertino Store',
      description: ' ',
      serialNumber: uuidv4(),
      ipAddress: '10.1.2.37',
      gateway: '10.1.2.1',
      subnet: '10.1.0.0',
      edgeDevices: 1,
      storageCapacity: 100,
      storageUsage: 5,
      connected: false,
    };
    const edgeId = edge.id;
    const edgeInfo: EdgeInfo = {
      tenantId,
      version,
      edgeId,
      id: edgeId,
      NumCPU: '4',
      TotalMemoryKB: '' + 16 * (1 << 20),
      TotalStorageKB: '' + 2 * (1 << 30),
      GPUInfo: 'NVIDIA',
      CPUUsage: '25.3',
      MemoryFreeKB: '' + 8 * (1 << 20),
      StorageFreeKB: '' + (1 << 30),
      GPUUsage: '15.6',
    };

    const user: User = {
      tenantId,
      version,
      id: uuidv4(),
      email: 'admin@smartretail.com',
      name: 'Admin',
      password: getSha256('apex'),
      role: UserRoleType.INFRA_ADMIN,
    };

    const project: Project = {
      tenantId,
      version,
      id: getDefaultProjectId(tenantId),
      name: 'Default Project',
      description: 'Default Project for backward compatibility',
      cloudCredentialIds: [cloudProfile.id],
      dockerProfileIds: [dockerProfile.id],
      users: [
        {
          userId: user.id,
          role: 'PROJECT_ADMIN',
        },
      ],
      edgeSelectorType: 'Explicit',
      edgeIds: [edge.id],
      edgeSelectors: [],
    };

    const dataSource: DataSource = {
      tenantId,
      version,
      edgeId: edge.id,
      id: uuidv4(),
      name: 'POS Camera 1',
      type: 'Sensor',
      sensorModel: '',
      connection: 'Secure',
      fields: [
        {
          name: 'Camera Feed',
          fieldType: 'Image',
          mqttTopic: 'rtsp://u@p:domain:554/topic',
        },
      ],
      selectors: [
        {
          id: catLocation.id,
          value: '1',
          scope: ['__ALL__'],
        },
      ],
      protocol: 'RTSP',
      authType: 'PASSWORD',
    };

    const scriptRuntime: ScriptRuntime = {
      tenantId,
      version,
      id: uuidv4(),
      name: 'My Node Env',
      description: 'My NodeJS Runtime',
      language: 'node',
      builtin: false,
      dockerRepoURI: 'node-env-2',
      dockerProfileID: dockerProfile.id,
      dockerfile: '',
      projectId: null,
    };

    const script: Script = {
      tenantId,
      version,
      id: uuidv4(),
      name: 'People counter',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'tensorflow-python',
      code: `def main: pass`,
      params: [],
      builtin: true,
      projectId: project.id,
      runtimeId: scriptRuntime.id,
      runtimeTag: '',
    };

    const dataStream: DataStream = {
      tenantId,
      version,
      id: uuidv4(),
      name: 'Counter 1 Counts',
      description: ' ',
      dataType: 'Image',
      origin: 'Data Source',
      originSelectors: [
        {
          id: catLocation.id,
          value: '1',
        },
      ],
      destination: DataStreamDestination.Edge,
      edgeStreamType: EdgeStreamType.ElasticSearch,
      size: 0,
      enableSampling: true,
      samplingInterval: 60,
      transformationArgsList: [],
      dataRetention: [],
      projectId: project.id,
      endPoint: 'datastream-endpoint',
    };

    const application: Application = {
      tenantId,
      version,
      id: uuidv4(),
      name: 'docker-flaskweb-hub',
      description: 'docker hub3docker hub3',
      // originSelectors: [],
      projectId: project.id,
      yamlData:
        'apiVersion: v1\nkind: Pod\nmetadata:\n  name: flask-web-server-waldot-repo\n  labels:\n    app: flask-web-server-waldot-repo\nspec:\n  containers:\n  - image: ntayal/myrepo:latest\n    name: flask-web-server-waldot-repo\n    imagePullPolicy: Always\n    ports:\n    - containerPort: 5000\n---\nkind: Service\napiVersion: v1\nmetadata:\n  name: flask-web-server-svc-waldot-repo\nspec:\n  selector:\n    app: flask-web-server-waldot-repo\n  ports:\n  - protocol: TCP\n    name: flask-web-server-waldot-repo\n    port: 5000\n    targetPort: 5000',
    };

    const applicationId = application.id;
    const applicationStatus: ApplicationStatus = {
      ...APP_STATUS,
      tenantId,
      edgeId,
      applicationId,
    };

    await tenantBuilder
      .addCategory(catLocation)
      .addUser(user)
      .addEdge(edge)
      .addScript(script)
      .addDataSource(dataSource)
      .addDataStream(dataStream)
      .addCloudProfile(cloudProfile)
      .addApplication(application)
      .addDockerProfile(dockerProfile)
      .addScriptRuntime(scriptRuntime)
      .addApplicationStatus(applicationStatus)
      .addProject(project)
      .addEdgeInfo(edgeInfo)
      .create();
  } catch (e) {
    console.log('Unexpected error:', e);
  }

  sql.close();
}

main();
