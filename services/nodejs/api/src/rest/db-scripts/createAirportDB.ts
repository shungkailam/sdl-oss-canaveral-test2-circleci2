import airport_codes from './airport_codes';
import { getMockIPAddress, getMockGateway, getMockSubnet } from './commonDB';
import { DocType, DocTypes } from '../model/baseModel';
import { staggerPromises } from './common';

import {
  TENANT_ID,
  TENANT_ID_2,
  MOCK_SCRIPTS,
  MOCK_DATA_STREAMS,
  EDGE_NAMES_SUBSET,
  AIRPORT_LOCATION_TYPE_VALUES,
  VIDEO_RESOLUTION_VALUES,
  getUser,
} from './dataDB';
import * as uuidv4 from 'uuid/v4';
import { CloudCreds, CloudType, Project } from '../model/index';
import { initSequelize, ignorePromiseError } from '../sql-api/baseApi';

import { getDBService, isSQL } from '../db-configurator/dbConfigurator';
import platformService from '../services/platform.service';
import { getVersion } from './createDataHelper';
import { createDocNew } from './tenantBuilder';
import { TenantBuilder } from './tenantBuilder';
import { createBuiltinScriptRuntimes } from './scriptRuntimeHelper';
import {
  createDataTypeCategory,
  addDataTypeCategorySelectors,
} from './categoryHelper';
import { deleteTenant, genEdgeInfo } from './common';

const startTime = Date.now();
const dbService = getDBService();
const { getAllDocuments } = dbService;
import {
  createDefaultProject,
  getDefaultProjectId,
} from '../../scripts/common';

const NOOP = () => {
  // no op
  return Promise.resolve();
};
function sleep(millis) {
  return new Promise(resolve => {
    setTimeout(() => {
      resolve(true);
    }, millis);
  });
}

// main function
// declare as async so we can use ES7 async/await
async function main() {
  let sql = initSequelize();

  console.log('Clearing existing tenant records');
  await deleteTenant(sql, TENANT_ID);

  console.log('Creating new tenant records');
  // TODO FIXME - do we need to wait for table creation?
  const version = getVersion();
  const TENANT_TOKEN = await platformService.getKeyService().genTenantToken();
  const tenant = {
    version,
    id: TENANT_ID,
    name: 'WalDot',
    token: TENANT_TOKEN,
    description: ' ',
  };
  const tenantBuilder = new TenantBuilder(tenant, sql);

  await createTenantFull(tenant, sql);

  console.log('DONE CREATE TENANT');

  sql.close();
}

async function createTenantFull(doc, sql) {
  // create tenant
  const tenantId = doc.id;
  const version = getVersion();

  // create tenant
  await createDocNew(tenantId, DocType.Tenant, doc);

  // create builtin script runtimes
  await createBuiltinScriptRuntimes(sql, tenantId);

  // create data type category
  const dtc = await createDataTypeCategory(sql, tenantId);
  console.log('DONE create data type cat:', dtc);

  let user = <any>getUser(tenantId);
  if (user) {
    user = { version, ...user };
    await createDocNew(tenantId, DocType.User, user);
  }

  const edgeDataList = [];
  // create edges for our tenant
  const edges = await Promise.all(
    airport_codes
      .filter(ac => EDGE_NAMES_SUBSET.indexOf(ac) !== -1)
      .map((ac, i) => {
        const cap = Math.round(400000 * Math.random()) / 100;
        let ac2 = ac;
        // differentiate edge id b/w tenant since id need to be unique
        if (tenantId !== 'tenant-id-waldot') {
          ac2 = `${ac}2`;
        }
        const edgeData = {
          tenantId,
          version,
          id: ac2,
          name: ac2,
          description: `Description for ${ac}`,
          serialNumber: uuidv4(),
          ipAddress: getMockIPAddress(i + 1),
          gateway: getMockGateway(i + 1),
          subnet: getMockSubnet(i + 1),
          edgeDevices: Math.floor(2 + 4 * Math.random()),
          storageCapacity: cap,
          storageUsage: Math.round(100 * Math.random() * cap) / 100,
          connected: false,
        };
        edgeDataList.push(edgeData);
        return createDocNew(tenantId, DocType.Edge, edgeData);
      })
  );

  console.log('>>> create edges: ', edges);

  const cloudCredsList: CloudCreds[] = [
    {
      version,
      id: uuidv4(),
      type: CloudType.AWS,
      name: 'AWS profile',
      description: 'mock AWS profile',
      awsCredential: JSON.stringify({ accessKey: 'foo', secret: 'bar' }),
      tenantId,
    },
    {
      version,
      id: uuidv4(),
      type: CloudType.GCP,
      name: 'GCP profile',
      description: 'mock GCP profile',
      gcpCredential: JSON.stringify({
        type: 'service_account',
        project_id: 'foo',
        private_key_id: 'foo',
        private_key: 'foo',
        client_email: 'foo',
        client_id: 'foo',
        auth_uri: 'foo',
        token_uri: 'foo',
        auth_provider_x509_cert_url: 'foo',
        client_x509_cert_url: 'foo',
      }),
      tenantId,
    },
  ];
  await Promise.all(
    cloudCredsList.map(cred => createDocNew(tenantId, DocType.CloudCreds, cred))
  );

  const airportCategoryId = uuidv4();
  const airportCategory = await createDocNew(tenantId, DocType.Category, {
    version,
    tenantId,
    id: airportCategoryId,
    name: 'Airport',
    purpose: 'empty purpose',
    values: EDGE_NAMES_SUBSET,
  });

  const airportLocationTypeCategoryId = uuidv4();
  const airportLocationTypeCategory = await createDocNew(
    tenantId,
    DocType.Category,
    {
      version,
      tenantId,
      id: airportLocationTypeCategoryId,
      name: 'Airport Location',
      purpose: 'empty purpose',
      values: AIRPORT_LOCATION_TYPE_VALUES,
    }
  );

  const videoResulotionCategoryId = uuidv4();
  const videoResulotionCategory = await createDocNew(
    tenantId,
    DocType.Category,
    {
      version,
      tenantId,
      id: videoResulotionCategoryId,
      name: 'Video Resolution',
      purpose: 'empty purpose',
      values: VIDEO_RESOLUTION_VALUES,
    }
  );

  // create default project
  console.log('creating default project for ' + tenantId);
  await createDefaultProject(sql, tenantId, false);

  const projectId = getDefaultProjectId(tenantId);

  const projectList: Project[] = [
    {
      version,
      id: uuidv4(),
      name: 'mock Project 1',
      description: 'mock Project #1 for testing',
      cloudCredentialIds: [cloudCredsList[0].id],
      dockerProfileIds: [],
      users: user ? [{ userId: user.id, role: 'PROJECT_ADMIN' }] : [],
      edgeSelectorType: 'Explicit',
      edgeIds: ['ORD'],
      edgeSelectors: [],
      tenantId,
    },
    {
      version,
      id: uuidv4(),
      name: 'mock Project 2',
      description: 'mock Project #2 for testing',
      cloudCredentialIds: [cloudCredsList[1].id],
      dockerProfileIds: [],
      users: [],
      edgeSelectorType: 'Category',
      edgeIds: [],
      edgeSelectors: [
        {
          id: airportCategoryId,
          value: EDGE_NAMES_SUBSET[0],
        },
        {
          id: airportCategoryId,
          value: EDGE_NAMES_SUBSET[1],
        },
      ],
      tenantId,
    },
  ];
  await Promise.all(
    projectList.map(proj => createDocNew(tenantId, DocType.Project, proj))
  );

  // create scripts for our tenant
  const scriptGenP = s => {
    const id = uuidv4();
    return createDocNew(tenantId, DocType.Script, {
      tenantId,
      version,
      id,
      description: ' ',
      projectId,
      ...s,
    });
  };

  const scripts = await staggerPromises(MOCK_SCRIPTS, scriptGenP, 10, NOOP);

  // create data streams for our tenant
  const datastreams_p1 = await Promise.all(
    MOCK_DATA_STREAMS.slice(0, 3).map(s => {
      const i1 = Math.floor(
        AIRPORT_LOCATION_TYPE_VALUES.length * Math.random()
      );
      s.originSelectors.push({
        id: airportLocationTypeCategoryId,
        value: 'Terminal',
      });
      s.originSelectors.push({
        id: airportLocationTypeCategoryId,
        value: 'Parking Lot',
      });
      s.originSelectors.push({
        id: airportLocationTypeCategoryId,
        value: 'Kiosk',
      });
      const id = uuidv4();
      return createDocNew(tenantId, DocType.DataStream, {
        id,
        tenantId,
        version,
        description: ' ',
        projectId,
        ...s,
      });
    })
  );
  const datastreams_p2 = await Promise.all(
    MOCK_DATA_STREAMS.slice(3).map((s, i) => {
      s.cloudCredsId = cloudCredsList[i].id;
      const i1 = Math.floor(
        AIRPORT_LOCATION_TYPE_VALUES.length * Math.random()
      );
      if (s.origin === 'Data Stream') {
        s['originId'] = datastreams_p1[s['originIndex']]._id;
      } else {
        s.originSelectors.push({
          id: airportLocationTypeCategoryId,
          value: 'Terminal',
        });
        s.originSelectors.push({
          id: airportLocationTypeCategoryId,
          value: 'Parking Lot',
        });
        s.originSelectors.push({
          id: airportLocationTypeCategoryId,
          value: 'Kiosk',
        });
      }
      const id = uuidv4();
      return createDocNew(tenantId, DocType.DataStream, {
        id,
        tenantId,
        version,
        description: ' ',
        projectId,
        ...s,
      });
    })
  );
  console.log('>>> datastreams: ', datastreams_p1.concat(datastreams_p2));

  const edgeInfos = edgeDataList.map(({ id: edgeId }) =>
    createDocNew(
      tenantId,
      DocType.EdgeInfo,
      genEdgeInfo({
        tenantId,
        version,
        edgeId,
        id: edgeId,
      })
    )
  );

  // for each edge, create data services, nodes, and sensors

  // For the calls to scale, must not call all createDocument upfront,
  // instead, should spread them across multiple await(s)
  let sensor_count = 0;
  const TOTAL_SENSORS = 2000;
  // forEach(async ...) is not waited for
  // 2000 sensors per edge
  edges.forEach(async edge => {
    const sensorGenFn = i => {
      const edgeId = edge._id;
      const topicName = `/nextNiceDemo/${edgeId}/Camera${i + 1}/image`;
      return createDocNew(tenantId, DocType.Sensor, {
        id: uuidv4(),
        tenantId,
        version,
        edgeId,
        topicName,
      });
    };
    await staggerPromises(
      Array(TOTAL_SENSORS)
        .fill(0)
        .map((x, i) => i),
      sensorGenFn,
      4,
      null
    );
  });

  console.log('>>> Done creating sensors');

  // Camera data source
  const datasourcesResult = <any[]>await Promise.all(
    edges.reduce((promises, edge) => {
      const edgeId = edge._id;
      const TERMINAL_CAMS = 1200;

      // note: elasticsearch can't handle too many (> 200) pending promises (default config?)
      promises = promises.concat(
        [edgeId].map((x_, i) => {
          let x = {
            name: `${x_}_Image_SRC`,
            type: 'Sensor',
            sensorModel: 'Model S',
            connection: 'Secure',
            fields: [],
            selectors: [],
            protocol: 'MQTT',
            authType: 'CERTIFICATE',
          };
          x.fields = Array(TOTAL_SENSORS)
            .fill(0)
            .map((xx, ii) => {
              return {
                name: `nextNiceDemo${x_}Feed${ii}`,
                mqttTopic: `/nextNiceDemo/${x_}/Camera${ii}/image`,
                fieldType: 'Image',
              };
            });
          x.selectors = [
            {
              id: airportCategoryId,
              value: x_,
              scope: ['__ALL__'],
            },
            {
              id: airportLocationTypeCategoryId,
              value: 'Terminal',
              scope: Array(TERMINAL_CAMS)
                .fill(0)
                .map((xx, ii) => `nextNiceDemo${x_}Feed${ii}`),
            },
            {
              id: airportLocationTypeCategoryId,
              value: 'Parking Lot',
              scope: Array(TOTAL_SENSORS - TERMINAL_CAMS)
                .fill(0)
                .map((xx, ii) => `nextNiceDemo${x_}Feed${TERMINAL_CAMS + ii}`),
            },
          ];
          addDataTypeCategorySelectors(tenantId, x.fields, x.selectors);
          const id = uuidv4();
          return createDocNew(tenantId, DocType.DataSource, {
            id,
            tenantId,
            version,
            edgeId,
            ...x,
          });
        })
      );
      return promises;
    }, [])
  );

  // Temperature data source
  const TEMP_SENSORS = 100;
  const datasourcesTemperatureResult = <any[]>await Promise.all(
    edges.reduce((promises, edge) => {
      const edgeId = edge._id;

      // note: elasticsearch can't handle too many (> 200) pending promises (default config?)
      promises = promises.concat(
        [edgeId].map((x_, i) => {
          let x = {
            name: `${x_}_Temp_SRC`,
            type: 'Sensor',
            sensorModel: 'Model 3',
            connection: 'Secure',
            fields: [],
            selectors: [],
            protocol: 'MQTT',
            authType: 'CERTIFICATE',
          };
          x.fields = Array(TEMP_SENSORS)
            .fill(0)
            .map((xx, ii) => {
              return {
                name: `nextNiceDemo${x_}_TempFeed${ii}`,
                mqttTopic: `/nextNiceDemo/${x_}/Thermometer${ii}/temp`,
                fieldType: 'Temperature',
              };
            });
          x.selectors = [
            {
              id: airportCategoryId,
              value: x_,
              scope: ['__ALL__'],
            },
            {
              id: airportLocationTypeCategoryId,
              value: 'Terminal',
              scope: ['__ALL__'],
            },
          ];
          addDataTypeCategorySelectors(tenantId, x.fields, x.selectors);
          const id = uuidv4();
          return createDocNew(tenantId, DocType.DataSource, {
            id,
            tenantId,
            version,
            edgeId,
            ...x,
          });
        })
      );
      return promises;
    }, [])
  );

  // Temperature data source
  const KIOSK_SENSORS = 250;
  const datasourcesKioskResult = <any[]>await Promise.all(
    edges.reduce((promises, edge) => {
      const edgeId = edge._id;

      // note: elasticsearch can't handle too many (> 200) pending promises (default config?)
      promises = promises.concat(
        [edgeId].map((x_, i) => {
          let x = {
            name: `${x_}_Kiosk_SRC`,
            type: 'Sensor',
            sensorModel: 'Model 3',
            connection: 'Secure',
            fields: [],
            selectors: [],
            protocol: 'MQTT',
            authType: 'CERTIFICATE',
          };
          x.fields = Array(KIOSK_SENSORS)
            .fill(0)
            .map((xx, ii) => {
              return {
                name: `nextNiceDemo${x_}_KioskFeed${ii}`,
                mqttTopic: `/nextNiceDemo/${x_}/Kiosk${ii}/json`,
                fieldType: 'Custom',
              };
            });
          x.selectors = [
            {
              id: airportCategoryId,
              value: x_,
              scope: ['__ALL__'],
            },
            {
              id: airportLocationTypeCategoryId,
              value: 'Lobby',
              scope: ['__ALL__'],
            },
          ];
          addDataTypeCategorySelectors(tenantId, x.fields, x.selectors);
          const id = uuidv4();
          return createDocNew(tenantId, DocType.DataSource, {
            id,
            tenantId,
            version,
            edgeId,
            ...x,
          });
        })
      );
      return promises;
    }, [])
  );

  console.log('DB setup done in ' + (Date.now() - startTime) + 'ms');

  return tenantId;
}

// helper to show DB document entries
// make this separate since ElasticSearch is NRT
// and we need to wait before issuing queries after index creation
async function query(tenantId) {
  const queryStartTime = Date.now();
  const docsList = await Promise.all(
    DocTypes.map(docType => getAllDocuments(tenantId, docType))
  );
  docsList.forEach(docs => console.log(docs));
  console.log('DB query done in ' + (Date.now() - queryStartTime) + 'ms');
}

main();
