import airport_codes from './airport_codes';
import { DocType } from '../model/baseModel';
import {
  createDocument,
  getAllDocs,
  refreshIndex,
  createTenant,
} from '../es-api/baseApi';
import {
  deleteIndices,
  waitForAllPromises,
  setupGlobalIndexMaybe,
  getMockIPAddress,
} from './commonDB';
import { GLOBAL_INDEX_NAME } from '../constants';
import {
  TENANT_TOKEN,
  TENANT_TOKEN_2,
  TENANT_ID,
  TENANT_ID_2,
  GLOBAL_INDEX_NAME_INTERNAL,
  MOCK_SCRIPTS,
  MOCK_DATA_SOURCES,
  MOCK_DATA_STREAMS,
  EDGE_NAMES_SUBSET,
  AIRPORT_LOCATION_TYPE_VALUES,
  VIDEO_RESOLUTION_VALUES,
  getUser,
} from './dataDB';
import * as uuidv4 from 'uuid/v4';
import { CloudCreds, CloudType } from '../model/index';

const startTime = new Date().getTime();

const NOOP = () => {
  // no op
  return Promise.resolve();
};

// create clouds
let aws, azure, gcp, xi;
const SKIP_REFRESH = true;

// main function
// declare as async so we can use ES7 async/await
async function main() {
  await deleteIndices();
  await setupGlobalIndexMaybe();

  const tid = await createTenantFull({
    id: TENANT_ID,
    name: 'WalDot',
    token: TENANT_TOKEN,
  });
  const tid2 = await createTenantFull({
    id: TENANT_ID_2,
    name: 'Rocket Blue',
    token: TENANT_TOKEN_2,
  });

  // wait is required since ElasticSearch is only Near Real Time (NRT)
  setTimeout(() => {
    query(tid);
    query(tid2);
  }, 2000);
}

async function createTenantFull(doc) {
  // create tenant
  const tenantId = (await createTenant(GLOBAL_INDEX_NAME, doc))._id;

  const user = getUser(tenantId);
  if (user) {
    await createDocument(tenantId, DocType.User, user);
  }

  const cloudCredsList: CloudCreds[] = [
    {
      id: uuidv4(),
      type: CloudType.AWS,
      name: 'AWS profile',
      description: 'mock AWS profile',
      awsCredential: JSON.stringify({ accessKey: 'foo', secret: 'bar' }),
      tenantId,
    },
    {
      id: uuidv4(),
      type: CloudType.GCP,
      name: 'GCP profile',
      description: 'mock GCP profile',
      gcpCredential: JSON.stringify({
        type: 'service_account',
        project_id: 'foo',
        private_key_id: '',
        private_key: '',
        client_email: '',
        client_id: '',
        auth_uri: '',
        token_uri: '',
        auth_provider_x509_cert_url: '',
        client_x509_cert_url: '',
      }),
      tenantId,
    },
  ];
  await Promise.all(
    cloudCredsList.map(cred =>
      createDocument(tenantId, DocType.CloudCreds, cred)
    )
  );

  const airportCategory = await createDocument(
    tenantId,
    DocType.Category,
    {
      tenantId,
      id: uuidv4(),
      name: 'Airport',
      purpose: '',
      values: EDGE_NAMES_SUBSET,
    },
    SKIP_REFRESH
  );

  const airportLocationTypeCategory = await createDocument(
    tenantId,
    DocType.Category,
    {
      tenantId,
      id: uuidv4(),
      name: 'Airport Location',
      purpose: '',
      values: AIRPORT_LOCATION_TYPE_VALUES,
    },
    SKIP_REFRESH
  );

  const videoResulotionCategory = await createDocument(
    tenantId,
    DocType.Category,
    {
      tenantId,
      id: uuidv4(),
      name: 'Video Resolution',
      purpose: '',
      values: VIDEO_RESOLUTION_VALUES,
    },
    SKIP_REFRESH
  );

  console.log('airport loc cat: ', airportLocationTypeCategory);

  // create scripts for our tenant
  const scripts = await waitForAllPromises(
    MOCK_SCRIPTS.map(s => {
      // use uuid for script id since fission endpoint must be of the form:
      //   [a-z0-9][-a-z0-9]*[a-z0-9]
      // but elasticsearch generated id may end with -
      const id = uuidv4();
      return createDocument(
        tenantId,
        DocType.Script,
        {
          tenantId,
          id,
          ...s,
        },
        SKIP_REFRESH
      );
    }),
    NOOP
  );

  // create data streams for our tenant
  const datastreams_p1 = await waitForAllPromises(
    MOCK_DATA_STREAMS.slice(0, 3).map(s => {
      const i1 = Math.floor(
        AIRPORT_LOCATION_TYPE_VALUES.length * Math.random()
      );
      s.originSelectors.push({
        id: airportLocationTypeCategory._id,
        value: 'Terminal',
      });
      s.originSelectors.push({
        id: airportLocationTypeCategory._id,
        value: 'Parking Lot',
      });
      s.originSelectors.push({
        id: airportLocationTypeCategory._id,
        value: 'Kiosk',
      });
      const id = uuidv4();
      return createDocument(
        tenantId,
        DocType.DataStream,
        {
          id,
          tenantId,
          ...s,
        },
        SKIP_REFRESH
      );
    }),
    NOOP
  );
  const datastreams_p2 = await waitForAllPromises(
    MOCK_DATA_STREAMS.slice(3).map(s => {
      const i1 = Math.floor(
        AIRPORT_LOCATION_TYPE_VALUES.length * Math.random()
      );
      if (s.origin === 'Data Stream') {
        s['originId'] = datastreams_p1[s['originIndex']]._id;
      } else {
        s.originSelectors.push({
          id: airportLocationTypeCategory._id,
          value: 'Terminal',
        });
        s.originSelectors.push({
          id: airportLocationTypeCategory._id,
          value: 'Parking Lot',
        });
        s.originSelectors.push({
          id: airportLocationTypeCategory._id,
          value: 'Kiosk',
        });
      }
      const id = uuidv4();
      return createDocument(
        tenantId,
        DocType.DataStream,
        {
          id,
          tenantId,
          ...s,
        },
        SKIP_REFRESH
      );
    }),
    NOOP
  );
  console.log('>>> datastreams: ', datastreams_p1.concat(datastreams_p2));

  // create edges for our tenant
  const edges = await waitForAllPromises(
    airport_codes
      .filter(ac => EDGE_NAMES_SUBSET.indexOf(ac) !== -1)
      .map((ac, i) => {
        const cap = Math.round(400000 * Math.random()) / 100;
        let ac2 = ac;
        // differentiate edge id b/w tenant since id need to be unique
        if (tenantId !== 'tenant-id-waldot') {
          ac2 = `${ac}2`;
        }
        return createDocument(
          tenantId,
          DocType.Edge,
          {
            tenantId,
            id: ac2,
            name: ac,
            description: `Description for ${ac}`,
            serialNumber: uuidv4(),
            ipAddress: getMockIPAddress(i + 1),
            edgeDevices: Math.floor(2 + 4 * Math.random()),
            storageCapacity: cap,
            storageUsage: Math.round(100 * Math.random() * cap) / 100,
          },
          SKIP_REFRESH
        );
      }),
    () => refreshIndex(tenantId)
  );

  console.log('>>> create edges: ', edges);
  // const node_count = Math.floor(2 + 4 * Math.random());
  // promises = promises.concat(
  //   Array(node_count)
  //     .fill(0)
  //     .map((x, i) =>
  //       createDoc(client, tenantId, 'node', {
  //         tenantId,
  //         edgeId,
  //         name: `N${i}`,
  //       })
  //     )
  // );

  // for each edge, create data services, nodes, and sensors

  // For the calls to scale, must not call all createDocument upfront,
  // instead, should spread them across multiple await(s)
  let sensor_count = 0;
  const TOTAL_SENSORS = 2000;
  edges.forEach(async edge => {
    const N = 20;
    const edgeId = edge._id;
    while (sensor_count < TOTAL_SENSORS) {
      await Promise.all(
        Array(N)
          .fill(0)
          .map((x, i) => {
            const topicName = `/nextNiceDemo/${edgeId}/Camera${i + 1}/image`;
            console.log('>>> creating sensor: ' + topicName);
            return createDocument(
              tenantId,
              DocType.Sensor,
              {
                tenantId,
                edgeId,
                topicName,
              },
              SKIP_REFRESH
            );
          })
      );
      sensor_count += N;
    }
  });

  console.log('>>> Done creating sensors');

  // Camera data source
  const datasourcesResult = <any[]>await waitForAllPromises(
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
              id: airportCategory._id,
              value: x_,
              scope: ['__ALL__'],
            },
            {
              id: airportLocationTypeCategory._id,
              value: 'Terminal',
              scope: Array(TERMINAL_CAMS)
                .fill(0)
                .map((xx, ii) => `nextNiceDemo${x_}Feed${ii}`),
            },
            {
              id: airportLocationTypeCategory._id,
              value: 'Parking Lot',
              scope: Array(TOTAL_SENSORS - TERMINAL_CAMS)
                .fill(0)
                .map((xx, ii) => `nextNiceDemo${x_}Feed${TERMINAL_CAMS + ii}`),
            },
          ];
          const id = uuidv4();
          return createDocument(
            tenantId,
            DocType.DataSource,
            {
              id,
              tenantId,
              edgeId,
              ...x,
            },
            SKIP_REFRESH
          );
        })
      );
      return promises;
    }, []),
    () => refreshIndex(tenantId)
  );

  // Temperature data source
  const TEMP_SENSORS = 100;
  const datasourcesTemperatureResult = <any[]>await waitForAllPromises(
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
              id: airportCategory._id,
              value: x_,
              scope: ['__ALL__'],
            },
            {
              id: airportLocationTypeCategory._id,
              value: 'Terminal',
              scope: ['__ALL__'],
            },
          ];
          const id = uuidv4();
          return createDocument(
            tenantId,
            DocType.DataSource,
            {
              id,
              tenantId,
              edgeId,
              ...x,
            },
            SKIP_REFRESH
          );
        })
      );
      return promises;
    }, []),
    () => refreshIndex(tenantId)
  );

  // Temperature data source
  const KIOSK_SENSORS = 250;
  const datasourcesKioskResult = <any[]>await waitForAllPromises(
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
              id: airportCategory._id,
              value: x_,
              scope: ['__ALL__'],
            },
            {
              id: airportLocationTypeCategory._id,
              value: 'Lobby',
              scope: ['__ALL__'],
            },
          ];
          const id = uuidv4();
          return createDocument(
            tenantId,
            DocType.DataSource,
            {
              id,
              tenantId,
              edgeId,
              ...x,
            },
            SKIP_REFRESH
          );
        })
      );
      return promises;
    }, []),
    () => refreshIndex(tenantId)
  );

  console.log('DB setup done in ' + (new Date().getTime() - startTime) + 'ms');

  return tenantId;
}

// helper to show DB document entries
// make this separate since ElasticSearch is NRT
// and we need to wait before issuing queries after index creation
async function query(tenantId) {
  const queryStartTime = new Date().getTime();
  console.log(await getAllDocs(tenantId, DocType.Tenant));
  // console.log((await getAllDocs(tenantId, 'cloud')));
  // console.log((await getAllDocs(tenantId, 'cloudservice')));
  console.log(await getAllDocs(tenantId, DocType.Edge));
  // console.log((await getAllDocs(tenantId, 'edgedataservice')));
  // console.log((await getAllDocs(tenantId, 'node')).hits.hits);
  console.log(await getAllDocs(tenantId, DocType.Sensor));
  console.log(await getAllDocs(tenantId, DocType.DataSource));
  // console.log((await getAllDocs(tenantId, 'project')));
  console.log(await getAllDocs(tenantId, DocType.DataStream));
  console.log(await getAllDocs(tenantId, DocType.Script));
  console.log(await getAllDocs(tenantId, DocType.Category));
  console.log(
    'DB query done in ' + (new Date().getTime() - queryStartTime) + 'ms'
  );
}

main();
