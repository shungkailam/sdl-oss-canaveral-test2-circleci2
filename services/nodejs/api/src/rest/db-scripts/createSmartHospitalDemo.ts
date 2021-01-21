// script to help create mock data for smart hospital
import { Tenant } from '../model/index';
import platformService from '../services/platform.service';
import { initSequelize } from '../sql-api/baseApi';
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
import { genEdgeInfo } from './common';
import { getDefaultProjectId } from '../../scripts/common';

async function main() {
  let sql = initSequelize();
  try {
    const version = getVersion();
    const tenantId = 'tid-demo-smart-hospital';
    const tenant: Tenant = {
      version,
      id: tenantId,
      name: 'Smart Hospital',
      token: await platformService.getKeyService().genTenantToken(),
    };
    const createDataHelper = new CreateDataHelper(tenant, sql);

    await createDataHelper.cleanDB();
    const user = createDataHelper.addUser(
      'Admin',
      'demo@smarthospital.com',
      'P@ssw0rd'
    );
    const edges = createDataHelper.addEdges([
      'Hospital-1',
      'Hospital-2',
      'Hospital-3',
      'Hospital-4',
      'Hospital-5',
      'Hospital-6',
      'Hospital-7',
      'Hospital-8',
      'Hospital-9',
      'Hospital-10',
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

    const ST_BED = 'Bed';
    const ST_HEART = 'Heart';
    const ST_CAMERA = 'Camera';
    const ST_TEMP = 'Temperature';

    const cats: any = {
      Cities: [
        'San Jose',
        'San Francisco',
        'New York',
        'Los Angeles',
        'Chicago',
        'Houston',
        'Phoenix',
        'Philadelphia',
        'San Antonio',
        'San Diego',
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
      SourceType: [ST_BED, ST_HEART, ST_CAMERA, ST_TEMP],
    };
    const categories = createDataHelper.addCategories(cats);
    // This must match the edges order category ID in selectors
    // First value belongs to category City and second to State
    const selectorValues = [
      ['San Jose', 'Floor-1'],
      ['San Jose', 'Floor-2'],
      ['San Francisco', 'Floor-1'],
      ['New York', 'Floor-1'],
      ['Los Angeles', 'Floor-1'],
      ['Chicago', 'Floor-1'],
      ['Houston', 'Floor-1'],
      ['Phoenix', 'Floor-1'],
      ['Philadelphia', 'Floor-1'],
      ['San Antonio', 'Floor-1'],
      ['San Diego', 'Floor-1'],
    ];

    // GCP account
    const gcpCloudProfile = createDataHelper.addCloudProfile({
      type: 'GCP',
      name: 'GCP Profile',
      description: 'Main GCP account profile',
      gcpCredential: JSON.stringify({
        type: 'service_account',
        project_id: 'foo',
      }),
    });
    // Default is AWS
    const cloudCreds = createDataHelper.addCloudProfile({});

    const dockerProfile = createDataHelper.addDockerProfile({});

    const projectId = getDefaultProjectId(tenantId);

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
            value: ST_CAMERA,
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
        name: '{{edge}}_Bed_SRC',
        protocol: 'RTSP',
        fields: [
          {
            name: 'bedfeed{{field}}',
            fieldType: 'Image',
            mqttTopic:
              'rtsp://bedadmin:bedpwd@10.15.232.{{ip}}/Streaming/Channels{{field}}/102',
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
            value: ST_BED,
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
            value: ST_TEMP,
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
        name: '{{edge}}_HEART_SRC',
        protocol: 'MQTT',
        fields: [
          {
            name: 'HEART_MON_{{field}}_{{edge}}',
            fieldType: 'Custom',
            mqttTopic: 'topic: /HEART/{{field}}',
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
            value: ST_HEART,
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
    const authorizedPersonnelScript = createDataHelper.addScript({
      name: 'Authorized personnel identification',
      description: ' ',
      type: 'Transformation',
      language: 'python',
      environment: 'tensorflow-python',
      code: faceMatchScript,
      params: [],
      runtimeId: `${tenantId}_sr-tensorflow`,
      projectId,
    });
    const patientSatisfactionScript = createDataHelper.addScript({
      name: 'Patient Satisfaction',
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
      ['San Jose', 'Floor-1'],
      ['San Jose', 'Floor-2'],
      ['San Francisco', 'Floor-1'],
      ['New York', 'Floor-1'],
      ['Los Angeles', 'Floor-1'],
      ['Chicago', 'Floor-1'],
      ['Houston', 'Floor-1'],
      ['Phoenix', 'Floor-1'],
      ['Philadelphia', 'Floor-1'],
      ['San Antonio', 'Floor-1'],
      ['San Diego', 'Floor-1'],
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
          value: ST_CAMERA,
        },
      ],
      destination: 'Cloud',
      cloudType: 'AWS',
      awsCloudRegion: 'us-west-2',
      awsStreamType: 'Kinesis',
      cloudCredsId: cloudCreds.id,
      transformationArgsList: [
        {
          args: [],
          transformationId: faceRecogScript.id,
        },
        {
          args: [],
          transformationId: authorizedPersonnelScript.id,
        },
      ],
      projectId,
    });
    createDataHelper.addDataStream(originSelectors, {
      name: 'Patient Satisfaction',
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
          value: ST_BED,
        },
      ],
      destination: 'Cloud',
      cloudType: 'AWS',
      awsCloudRegion: 'us-west-2',
      awsStreamType: 'Kinesis',
      cloudCredsId: cloudCreds.id,
      transformationArgsList: [
        {
          args: [],
          transformationId: faceRecogScript.id,
        },
        {
          args: [],
          transformationId: patientSatisfactionScript.id,
        },
      ],
      projectId,
    });
    const application = createDataHelper.addApplication({
      projectId,
      edgeIds: edges.slice(0, 5).map(edge => edge.id),
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
