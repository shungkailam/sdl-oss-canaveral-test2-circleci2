// script to help create tenant mock data

import { Tenant } from '../model/index';
import { DocType } from '../model/baseModel';
import { getSha256 } from '../util/cryptoUtil';
import * as uuidv4 from 'uuid/v4';
import platformService from '../services/platform.service';
import { initSequelize } from '../sql-api/baseApi';
import { isSQL } from '../db-configurator/dbConfigurator';
import { TenantBuilder } from './tenantBuilder';
import { searchObjects } from '../es-api/baseApi';
import { CreateDataHelper, getVersion } from './createDataHelper';
import {
  createDefaultProject,
  getDefaultProjectId,
} from '../../scripts/common';
import {
  faceRecogitionScript,
  faceMatchScript,
  customDataMoverScript,
  dataExtractionScript,
  imageProcessingScript,
  objectRecognitionScript,
  simpleAppScript,
  temperatureScript,
} from './applicationScripts';
import { Sequelize } from 'sequelize-typescript';
import { genEdgeInfo } from './common';

function random(min: number, max: number): number {
  return Math.ceil(Math.random() * (max - min) + min);
}

async function main() {
  let sql: Sequelize = initSequelize();
  try {
    const version = getVersion();
    const tenantId = 'tenant-id-numart-stores';
    const tenant: Tenant = {
      version,
      id: tenantId,
      name: 'nuMart Stores',
      token: await platformService.getKeyService().genTenantToken(),
    };
    const createDataHelper = new CreateDataHelper(tenant, sql);
    await createDataHelper.cleanDB();
    const user = createDataHelper.addUser(
      'Satyam Vaghani',
      'satyam@numart.com',
      'apex'
    );
    const edges = createDataHelper.addEdges([
      'San_Francisco',
      'Los_Angeles',
      'New_York',
      'Washington_DC',
      'New_Orleans',
      'Miami',
      'Seattle',
      'Portland',
      'Houston',
      'Salt_Lake_City',
    ]);
    const edgeInfos = edges.map(({ id: edgeId }) =>
      createDataHelper.addEdgeInfo(
        genEdgeInfo({
          tenantId,
          version,
          edgeId,
          id: edgeId,
        })
      )
    );
    const cats: any = {
      City: [
        'San_Francisco',
        'New_York',
        'Washington_DC',
        'New_Orleans',
        'Miami',
        'Seattle',
        'Portland',
        'Los_Angeles',
        'Houston',
        'Salt_Lake_City',
      ],
      State: [
        'California',
        'Florida',
        'Oregon',
        'New_York',
        'Washington_DC',
        'Washington',
        'Louisiana',
        'Florida',
        'Texas',
        'Utah',
      ],
    };
    const categories = createDataHelper.addCategories(cats);

    // GCP account
    const gcpCloudProfile = createDataHelper.addCloudProfile({
      type: 'GCP',
      name: 'GCP Profile',
      description: 'Main GCP account profile',
      gcpCredential: {
        type: 'service_account',
        project_id: 'foo',
      },
    });
    // Default is AWS
    const cloudCreds = createDataHelper.addCloudProfile({});

    const dockerProfile = createDataHelper.addDockerProfile({});

    // default project id
    // default project will be created inside createDataHelper.create()
    const projectId = getDefaultProjectId(tenantId);

    // This must match the edges order category ID in selectors
    // First value belongs to category City and second to State
    const selectorValues = [
      ['San_Francisco', 'California'],
      ['Los_Angeles', 'California'],
      ['New_York', 'New_York'],
      ['Washington_DC', 'Washington_DC'],
      ['New_Orleans', 'Louisiana'],
      ['Miami', 'Florida'],
      ['Seattle', 'Washington'],
      ['Portland', 'Oregon'],
      ['Houston', 'Texas'],
      ['Salt_Lake_City', 'Utah'],
    ];
    const dataSources = createDataHelper.addDataSources(
      selectorValues,
      edges,
      {
        name: '{{edge}}_Image_SRC',
        protocol: 'RTSP',
        fields: [
          {
            name: 'camfeed{{field}}',
            fieldType: 'Image',
            mqttTopic:
              'rtsp://camadmin:campwd@10.15.232.{{ip}}/Streaming/Channels{{field}}/102',
          },
        ],
        selectors: [
          {
            id: categories[0].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
          {
            id: categories[1].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
        ],
      },
      edgeName => {
        if (edgeName == 'San_Francisco') {
          return 10;
        }
        return random(5, 10);
      }
    );
    createDataHelper.addDataSources(
      selectorValues,
      edges,
      {
        name: '{{edge}}_HVAC_SRC',
        protocol: 'MQTT',
        fields: [
          {
            name: 'KIOSK_{{field}}_{{edge}}',
            fieldType: 'Temperature',
            mqttTopic: 'topic: /HVAC/{{field}}',
          },
        ],
        selectors: [
          {
            id: categories[0].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
          {
            id: categories[1].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
        ],
      },
      edgeName => {
        if (edgeName == 'San_Francisco') {
          return 20;
        }
        return random(15, 20);
      }
    );
    createDataHelper.addDataSources(
      selectorValues,
      edges,
      {
        name: '{{edge}}_KIOSK_SRC',
        protocol: 'MQTT',
        fields: [
          {
            name: 'KIOSK_{{field}}_{{edge}}',
            fieldType: 'Custom',
            mqttTopic: 'topic: /Kiosk/{{field}}',
          },
        ],
        selectors: [
          {
            id: categories[0].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
          {
            id: categories[1].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
        ],
      },
      edgeName => {
        if (edgeName === 'San_Francisco') {
          return 25;
        }
        return random(20, 25);
      }
    );
    const scriptRuntime = createDataHelper.addScriptRuntime({
      projectId,
    });
    createDataHelper.addScript({
      name: 'Node test',
      description: ' ',
      type: 'Transformation',
      code: ' ',
      params: [],
      runtimeId: scriptRuntime.id,
      projectId,
    });
    createDataHelper.addScript({
      name: 'Sampling',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'python-env',
      code: ' ',
      params: [],
      runtimeId: `${tenantId}_sr-python`,
      projectId,
    });
    createDataHelper.addScript({
      name: 'Custom Data Mover',
      description: ' ',
      type: 'Function',
      language: 'python',
      environment: 'python-env',
      code: customDataMoverScript,
      params: [],
      runtimeId: `${tenantId}_sr-python`,
      projectId,
    });
    createDataHelper.addScript({
      name: 'Data Extraction',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'python-env',
      code: dataExtractionScript,
      params: [],
      runtimeId: `${tenantId}_sr-python`,
      projectId,
    });
    createDataHelper.addScript({
      name: 'Image Processing',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'python-env',
      code: imageProcessingScript,
      params: [],
      runtimeId: `${tenantId}_sr-python`,
      projectId,
    });
    createDataHelper.addScript({
      name: 'Object Recognition',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'python-env',
      code: objectRecognitionScript,
      params: [],
      runtimeId: `${tenantId}_sr-python`,
      projectId,
    });
    createDataHelper.addScript({
      name: 'Simple App',
      description: ' ',
      type: 'Function',
      language: 'python',
      environment: 'python-env',
      code: simpleAppScript,
      params: [],
      runtimeId: `${tenantId}_sr-python`,
      projectId,
    });
    const tempScript = createDataHelper.addScript({
      name: 'Temperature',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'python-env',
      code: temperatureScript,
      params: [],
      runtimeId: `${tenantId}_sr-python`,
      projectId,
    });
    const faceRecogScript = createDataHelper.addScript({
      name: 'Face recognition',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'tensorflow-python',
      code: faceRecogitionScript,
      params: [],
      runtimeId: `${tenantId}_sr-tensorflow`,
      projectId,
    });
    const loyaltyMemberScript = createDataHelper.addScript({
      name: 'Loyalty member identification',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'tensorflow-python',
      code: faceMatchScript,
      params: [],
      runtimeId: `${tenantId}_sr-tensorflow`,
      projectId,
    });

    const originSelectors = [
      ['California'],
      ['Florida'],
      ['Oregon'],
      ['New_York'],
      ['Washington_DC'],
      ['Washington'],
      ['Louisiana'],
      ['Florida'],
      ['Texas'],
      ['Utah'],
    ];
    createDataHelper.addDataStream(originSelectors, {
      name: 'Temperature Monitoring Feed',
      description: ' ',
      originSelectors: [
        {
          id: categories[1].id,
          value: '{{selectorValue}}',
        },
      ],
      destination: 'Edge',
      edgeStreamType: 'None',
      transformationArgsList: [
        {
          args: [],
          transformationId: tempScript.id,
        },
      ],
      projectId,
    });
    createDataHelper.addDataStream(originSelectors, {
      name: 'Surveillance Feed',
      description: ' ',
      originSelectors: [
        {
          id: categories[1].id,
          value: '{{selectorValue}}',
        },
      ],
      destination: 'Cloud',
      cloudType: 'AWS',
      awsCloudRegion: 'us-west-2',
      awsStreamType: 'DynamoDB',
      cloudCredsId: cloudCreds.id,
      transformationArgsList: [
        {
          args: [],
          transformationId: faceRecogScript.id,
        },
        {
          args: [],
          transformationId: loyaltyMemberScript.id,
        },
      ],
      projectId,
    });

    const application = createDataHelper.addApplication({
      projectId,
    });
    const applicationId = application.id;
    const applicationStatuses = edges.slice(0, 5).map(edge =>
      createDataHelper.addApplicationStatus({
        applicationId,
        edgeId: edge.id,
      })
    );

    await createDataHelper.create();
  } catch (e) {
    console.log('Unexpected error:', e);
  }

  sql.close();
}

main();
