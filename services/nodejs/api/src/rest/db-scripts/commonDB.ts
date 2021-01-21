import { deleteIndex, createDocument, getESClient } from '../es-api/baseApi';
import { GLOBAL_INDEX_NAME_INTERNAL, TENANT_ID, TENANT_ID_2 } from './dataDB';
import { GLOBAL_INDEX_NAME } from '../constants';
import { DocType } from '../model/baseModel';

export async function deleteIndices() {
  await deleteIndex(GLOBAL_INDEX_NAME_INTERNAL);
  await deleteIndex(TENANT_ID);
  await deleteIndex(TENANT_ID_2);
}

export async function setupGlobalIndexMaybe() {
  const esClient = getESClient();
  try {
    const idx_exists = await esClient.indices.exists({
      index: GLOBAL_INDEX_NAME_INTERNAL,
    });
    if (idx_exists) {
      console.log(
        `>>> setupGlobalIndexMaybe - index [${GLOBAL_INDEX_NAME_INTERNAL}] exists, skip`
      );
      return Promise.resolve();
    }
  } catch (e) {
    // ignore
  }

  // Create index
  return esClient.indices
    .create({
      index: GLOBAL_INDEX_NAME_INTERNAL,
    })
    .then(() => {
      // set up alias
      return esClient.indices.putAlias({
        index: GLOBAL_INDEX_NAME_INTERNAL,
        name: GLOBAL_INDEX_NAME,
      });
    })
    .then(() => {
      // set up mappings for ids
      return esClient.indices.putMapping({
        index: GLOBAL_INDEX_NAME,
        type: 'ids',
        body: {
          properties: {
            id: {
              type: 'string',
              index: 'not_analyzed',
            },
            tenantId: {
              type: 'string',
              index: 'not_analyzed',
            },
            edgeId: {
              type: 'string',
              index: 'not_analyzed',
            },
            originId: {
              type: 'string',
              index: 'not_analyzed',
            },
            // transformationIds: {
            //   type: 'string',
            //   index: 'not_analyzed',
            // },
            type: {
              type: 'string',
              index: 'not_analyzed',
            },
            fieldType: {
              type: 'string',
              index: 'not_analyzed',
            },
            streamType: {
              type: 'string',
              index: 'not_analyzed',
            },
            token: {
              type: 'string',
              index: 'not_analyzed',
            },
            code: {
              type: 'string',
              index: 'not_analyzed',
            },
            origin: {
              type: 'string',
              index: 'not_analyzed',
            },
            destination: {
              type: 'string',
              index: 'not_analyzed',
            },
            scope: {
              type: 'string',
              index: 'not_analyzed',
            },
            topicName: {
              type: 'string',
              index: 'not_analyzed',
            },
            mqttTopic: {
              type: 'string',
              index: 'not_analyzed',
            },
            serialNumber: {
              type: 'string',
              index: 'not_analyzed',
            },
            cloud_id: {
              type: 'string',
              index: 'not_analyzed',
            },
            project_id: {
              type: 'string',
              index: 'not_analyzed',
            },
            email: {
              type: 'string',
              index: 'not_analyzed',
            },
            password: {
              type: 'string',
              index: 'not_analyzed',
            },
            size: {
              type: 'double',
            },
            storageCapacity: {
              type: 'double',
            },
            storageUsage: {
              type: 'double',
            },
          },
        },
      });
    })
    .then(() => {
      return esClient.indices.putMapping({
        index: GLOBAL_INDEX_NAME,
        type: 'datasource',
        body: {
          properties: {
            name: {
              type: 'string',
            },
            type: {
              type: 'string',
              index: 'not_analyzed',
            },
            sensorModel: {
              type: 'string',
              index: 'not_analyzed',
            },
            connection: {
              type: 'string',
              index: 'not_analyzed',
            },
            fields: {
              type: 'nested',
              properties: {
                name: {
                  type: 'string',
                },
                mqttTopic: {
                  type: 'string',
                  index: 'not_analyzed',
                },
                fieldType: {
                  type: 'string',
                  index: 'not_analyzed',
                },
              },
            },
            selectors: {
              type: 'nested',
              properties: {
                id: {
                  type: 'string',
                  index: 'not_analyzed',
                },
                value: {
                  type: 'string',
                },
                scope: {
                  type: 'string',
                  index: 'not_analyzed',
                },
              },
            },
          },
        },
      });
    })
    .then(() => {
      return esClient.indices.putMapping({
        index: GLOBAL_INDEX_NAME,
        type: 'datastream',
        body: {
          properties: {
            name: {
              type: 'string',
            },
            dataType: {
              type: 'string',
              index: 'not_analyzed',
            },
            origin: {
              type: 'string',
              index: 'not_analyzed',
            },
            originSelectors: {
              type: 'nested',
              properties: {
                id: {
                  type: 'string',
                  index: 'not_analyzed',
                },
                value: {
                  type: 'string',
                  index: 'not_analyzed',
                },
              },
            },
            originId: {
              type: 'string',
              index: 'not_analyzed',
            },
            destination: {
              type: 'string',
              index: 'not_analyzed',
            },
            streamType: {
              type: 'string',
              index: 'not_analyzed',
            },
            size: {
              type: 'double',
            },
            enableSampling: {
              type: 'boolean',
            },
            // transformationIds: {
            //   type: 'string',
            //   index: 'not_analyzed',
            // },
            samplingInterval: {
              type: 'long',
            },
            dataRetention: {
              type: 'nested',
              properties: {
                type: {
                  type: 'string',
                  index: 'not_analyzed',
                },
                limit: {
                  type: 'double',
                },
              },
            },
          },
        },
      });
    })
    .then(() => {
      return esClient.indices.refresh({ index: GLOBAL_INDEX_NAME_INTERNAL });
    });
}

export function getMockIPAddress(i) {
  return `10.5.72.${i}`;
}
export function getMockSubnet(i) {
  return `10.5.0.0`;
}
export function getMockGateway(i) {
  return `10.5.0.1`;
}

export async function waitForAllPromises(promises: Promise<any>[], callback) {
  console.log('>>> wait for all promises, count=' + promises.length);
  const N = 20;
  let results = [];
  if (promises.length <= N) {
    results = await Promise.all(promises);
    if (callback) {
      await callback();
    }
  } else {
    const n = Math.floor((promises.length + N - 1) / N);
    for (let i = 0; i < n; i++) {
      const ps = promises.slice(i * N, (i + 1) * N);
      const psa = await Promise.all(ps);
      console.log('>>> psa.length = ' + psa.length);
      results = results.concat(psa);
      console.log('>>> results.length = ' + results.length);
      // wait for half a sec
      if (i < n - 1) {
        console.log(
          'waitForAllPromises: cursor=' + (i + 1) * N + ', wait 2 sec'
        );

        if (callback) {
          await callback();
        }
      }
    }
  }
  console.log('>>> wait for all promises, result count=' + results.length);
  return results;
}

// not currently used
async function createProjectsAndDataStreamsForTenant(tenantId, doc) {
  // create projects
  const project_1 = await createDocument(tenantId, DocType.Project, {
    tenantId,
    name: `airport-dev-${doc.name}`,
    dataType: 'Camera, HVAC, Kiosk',
    sensors: 23500,
    edgeAllocation: '2.7 PB',
    cloudAllocation: '140.0 TB',
    cloudDestination: 'Google Cloud Storage',
  });
  const project_2 = await createDocument(tenantId, DocType.Project, {
    tenantId,
    name: `airport-analyst-${doc.name}`,
    dataType: 'Camera',
    sensors: 20000,
    edgeAllocation: '0.0 GB',
    cloudAllocation: '90.0 TB',
    cloudDestination: 'Microsoft Azure',
  });
  const project_3 = await createDocument(tenantId, DocType.Project, {
    tenantId,
    name: `airport-manager-${doc.name}`,
    dataType: 'HVAC, Kiosk',
    sensors: 3500,
    edgeAllocation: '500 TB',
    cloudAllocation: '-',
    cloudDestination: '-',
  });

  // create data streams for projects
  const project_id = project_1._id;
  const data_stream_1 = await createDocument(tenantId, DocType.DataStream, {
    tenantId,
    project_id,
    name: `security-camera-video-${doc.name}`,
    dataType: 'Camera',
    dataSource: 'security-cameras (150000 devices)',
    samplingRate: '1 Second',
    dataRetention: '1 Week',
    destination: 'Edge',
  });
  const data_stream_2 = await createDocument(tenantId, DocType.DataStream, {
    tenantId,
    project_id,
    name: `POI-edge-${doc.name}`,
    dataType: 'Camera',
    dataSource: 'POI-detectors (5000 devices)',
    samplingRate: '15 Seconds',
    dataRetention: 'Up to 200.0 TB',
    destination: 'Edge',
  });
  const data_stream_3 = await createDocument(tenantId, DocType.DataStream, {
    tenantId,
    project_id,
    name: `POI-cloud-${doc.name}`,
    dataType: 'Camera',
    dataSource: 'POI-edge',
    samplingRate: '1 Minute',
    dataRetention: 'Up to 60.0 TB',
    destination: 'Google Cloud Storage',
  });
  const data_stream_4 = await createDocument(tenantId, DocType.DataStream, {
    tenantId,
    project_id,
    name: `passenger-checkin-local-${doc.name}`,
    dataType: 'Kiosk',
    dataSource: 'gate-kiosks (2500 devices)',
    samplingRate: 'All',
    dataRetention: '8 Months',
    destination: 'Edge',
  });
  const data_stream_5 = await createDocument(tenantId, DocType.DataStream, {
    tenantId,
    project_id,
    name: `passenger-traffic-training-${doc.name}`,
    dataType: 'Processed',
    dataSource: 'passenger-checkin-local',
    samplingRate: 'Custom',
    dataRetention: 'Up to 20.0 TB',
    destination: 'Google Cloud Storage',
  });
}
