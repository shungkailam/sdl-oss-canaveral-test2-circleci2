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

function random(min: number, max: number): number {
  return Math.ceil(Math.random() * (max - min) + min);
}

async function main() {
  let sql = initSequelize();
  try {
    const version = getVersion();
    const tenantId = 'tenant-id-nuSec';
    const tenant: Tenant = {
      version,
      id: tenantId,
      name: 'Smart Security',
      token: await platformService.getKeyService().genTenantToken(),
    };
    const createDataHelper = new CreateDataHelper(tenant, sql);

    await createDataHelper.cleanDB();
    const user = createDataHelper.addUser(
      'Satyam Vaghani',
      'sherlock@nutanix.com',
      'nutanix/4u'
    );
    const edges = createDataHelper.addEdges([
      'SAN-JOSE-1',
      'SAN-JOSE-2',
      'SAN-FRANCISCO',
      'DURHAM',
      'BANGALORE',
      'SEATTLE',
      'BRISBANE',
      'SAO-PAULO',
      'BEIJING',
      'PARIS',
      'TOKYO',
    ]);
    const cats: any = {
      Countries: [
        'USA',
        'INDIA',
        'CHINA',
        'BRAZIL',
        'FRANCE',
        'JAPAN',
        'AUSTRALIA',
      ],
      Cities: [
        'San Jose',
        'San Francisco',
        'Durham',
        'Bangalore',
        'Seattle',
        'Brisbane',
        'Sao Paulo',
        'Beijing',
        'Paris',
        'Tokyo',
      ],
      Floors: [
        'Floor-1',
        'Floor-2',
        'Floor-3',
        'Floor-4',
        'Floor-5',
        'Floor-6',
        'Floor-7',
        'Floor-8',
        'Floor-9',
        'Floor-10',
      ],
    };
    const categories = createDataHelper.addCategories(cats);
    // This must match the edges order category ID in selectors
    // First value belongs to category City and second to State
    const selectorValues = [
      ['USA', 'San Jose', 'Floor-1'],
      ['USA', 'San Jose', 'Floor-2'],
      ['USA', 'San Francisco', 'Floor-1'],
      ['USA', 'Durham', 'Floor-1'],
      ['INDIA', 'Bangalore', 'Floor-1'],
      ['USA', 'Seattle', 'Floor-1'],
      ['AUSTRALIA', 'Brisbane', 'Floor-1'],
      ['BRAZIL', 'Sao Paulo', 'Floor-1'],
      ['CHINA', 'Beijing', 'Floor-1'],
      ['FRANCE', 'Paris', 'Floor-1'],
      ['JAPAN', 'Tokyo', 'Floor-1'],
    ];

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

    const project = createDataHelper.addProject({
      name: 'Default Project',
      description: 'default project',
      cloudCredentialIds: [gcpCloudProfile.id, cloudCreds.id],
      dockerProfileIds: [dockerProfile.id],
      users: [
        {
          userId: user.id,
          role: 'PROJECT_ADMIN',
        },
      ],
      edgeSelectorType: 'Explicit',
      edgeIds: edges.slice(0, 5).map(edge => edge.id),
      edgeSelectors: [],
    });
    const projectId = project.id;

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
          {
            id: categories[2].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
        ],
      },
      edgeName => {
        return 10;
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
          {
            id: categories[2].id,
            value: '{{selectorValue}}',
            scope: ['__ALL__'],
          },
        ],
      },
      edgeName => {
        return 10;
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
      ['USA', 'San Jose', 'Floor-1'],
      ['USA', 'San Jose', 'Floor-2'],
      ['USA', 'San Francisco', 'Floor-1'],
      ['USA', 'Durham', 'Floor-1'],
      ['INDIA', 'Bangalore', 'Floor-1'],
      ['USA', 'Seattle', 'Floor-1'],
      ['AUSTRALIA', 'Brisbane', 'Floor-1'],
      ['BRAZIL', 'Sao Paulo', 'Floor-1'],
      ['CHINA', 'Beijing', 'Floor-1'],
      ['FRANCE', 'Paris', 'Floor-1'],
      ['JAPAN', 'Tokyo', 'Floor-1'],
    ];
    createDataHelper.addDataStream(originSelectors, {
      name: 'Temperature Monitoring Feed',
      description: ' ',
      originSelectors: [
        {
          id: categories[0].id,
          value: '{{selectorValue}}',
        },
        {
          id: categories[1].id,
          value: '{{selectorValue}}',
        },
        {
          id: categories[2].id,
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
    });
    createDataHelper.addDataStream(originSelectors, {
      name: 'Surveillance Feed',
      description: ' ',
      originSelectors: [
        {
          id: categories[0].id,
          value: '{{selectorValue}}',
        },
        {
          id: categories[1].id,
          value: '{{selectorValue}}',
        },
        {
          id: categories[2].id,
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
