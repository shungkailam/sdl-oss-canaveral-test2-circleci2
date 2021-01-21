import { Client } from 'elasticsearch';
import * as omit from 'object.omit';
import { modelFromEs, modelToEs } from '../util/esUtil';
import {
  UpdateDocumentResponse,
  CreateDocumentResponse,
  DeleteDocumentResponse,
} from '../model/baseModel';
import { GetResponse } from 'elasticsearch';
import { GLOBAL_INDEX_NAME } from '../constants';
import { Tenant } from '../model/tenant';
import { Edge } from '../model/edge';
import { User } from '../model/user';
import { DocType } from '../model/baseModel';
import { logger } from '../util/logger';

const dbHost = process.env.ELASTICSEARCH_SERVICE_HOST || 'localhost';
const dbPort = process.env.ELASTICSEARCH_SERVICE_PORT || 9200;

// create client for ElasticSearch
// this is the main interface to ElasticSearch client API
// Do this lazily so we don't create it when DynamoDB is used instead of ES
let esClient: Client = null;
export function getESClient(): Client {
  if (!esClient) {
    esClient = new Client({
      host: `${dbHost}:${dbPort}`,
      log: 'info',
    });
  }
  return esClient;
}

/**
 * Delete the document with the given id, type for the tenant.
 * @param tenantId
 * @param id
 * @param type
 * @returns {Promise<Elasticsearch.DeleteDocumentResponse>}
 * @throws Will throw if not found
 */
export function deleteDocument(
  tenantId: string,
  id: string,
  type: DocType
): Promise<DeleteDocumentResponse> {
  return <any>esClient.delete({
    id,
    type,
    index: tenantId,
    refresh: true,
  });
}

/**
 * Get the document with the given id, type for the tenant.
 * @param tenantId
 * @param id
 * @param type
 * @returns {Promise<T>} where T is the model for the doc type
 * @throws Will throw if not found
 */
export async function getDocument<T>(
  tenantId: string,
  id: string,
  type: DocType
): Promise<T> {
  const resp: GetResponse<T> = await esClient.get<T>({
    id,
    type,
    index: tenantId,
  });
  return modelFromEs(resp);
}

/**
 * Update the document with the given id, type for the tenant.
 * @param tenantId
 * @param id
 * @param type
 * @param doc
 * @returns {Promise<UpdateDocumentResponse>}
 * @throws Will throw if not found
 */
export async function updateDocument(
  tenantId: string,
  id: string,
  type: DocType,
  doc
): Promise<UpdateDocumentResponse> {
  const esDoc = modelToEs(doc);
  return esClient.update({
    id,
    type,
    body: {
      doc: esDoc,
    },
    index: tenantId,
    refresh: true,
  });
}

/**
 * Create a document with the given type for the tenant.
 * If body.id exists, will use update with upsert,
 * else will use index call to create.
 * @param tenantId
 * @param type
 * @param body {T} where T is the model for type
 * @returns {Promise<Elasticsearch.CreateDocumentResponse>}
 */
export function createDocument(
  tenantId: string,
  type: DocType,
  body,
  skipRefresh?: boolean
): Promise<CreateDocumentResponse> {
  // Note: method much slower if set refresh to true
  // When inserting mock data we should set refresh to false
  //
  // if id exists, use update with upsert=true
  if (body.id) {
    const { id, ...doc } = body;
    let payload: any = {
      type,
      id,
      index: tenantId,
      body: {
        doc,
        doc_as_upsert: true,
      },
    };
    if (!skipRefresh) {
      payload.refresh = true;
    }
    return esClient.update(payload);
  } else {
    // id does not exist, use index to create
    let payload: any = {
      type,
      body,
      index: tenantId,
    };
    if (!skipRefresh) {
      payload.refresh = 'true';
    }
    return esClient.index(payload);
  }
}

async function getAllTenants(): Promise<Tenant[]> {
  const resp: any = await esClient.search({
    body: {
      query: { type: { value: DocType.Tenant } },
    },
    index: GLOBAL_INDEX_NAME,
    version: true,
  });
  const tenants: any[] = resp.hits.hits;
  return tenants.map(tenant => modelFromEs(tenant));
}

export async function getAllDocuments<T>(
  tenantId: string,
  type: DocType
): Promise<T[]> {
  if (type === DocType.Tenant) {
    const tenants: any = <any>await getAllTenants();
    logger.info('Get all tenants returns:', tenants);
    return tenants;
  }
  return searchObjects<T>({
    body: {
      query: {
        bool: {
          must: [{ match: { _type: type } }, { term: { tenantId } }],
        },
      },
      // TODO: implement paging support
      size: 200,
    },
    index: tenantId,
  });
}

export async function getAllDocumentsForEdge<T>(
  tenantId: string,
  edgeId: string,
  type: DocType
): Promise<T[]> {
  const ans = await searchObjects<T>({
    body: {
      query: {
        bool: {
          must: [
            { match: { _type: type } },
            { term: { tenantId } },
            { term: { edgeId } },
          ],
        },
      },
      // TODO: implement paging support
      size: 200,
    },
    index: tenantId,
  });
  return ans;
}

export async function getAggregate(
  tenantId: string,
  type: DocType,
  fieldName: string
): Promise<any> {
  const aggregateName = 'aggregate_name';
  const response = await esClient.search({
    type,
    index: tenantId,
    body: {
      size: 0,
      aggregations: {
        [aggregateName]: {
          terms: {
            field: fieldName,
            size: 200,
          },
        },
      },
    },
  });
  return response.aggregations[aggregateName].buckets;
}

export async function getNestedAggregate(
  tenantId: string,
  type: DocType,
  fieldName: string,
  nestedFieldName: string
): Promise<any> {
  const agg1 = 'agg1';
  const agg2 = 'agg2';
  const response = await esClient.search({
    type,
    index: tenantId,
    body: {
      size: 0,
      aggs: {
        [agg1]: {
          nested: {
            path: fieldName,
          },
          aggs: {
            [agg2]: {
              terms: {
                field: `${fieldName}.${nestedFieldName}`,
                size: 200,
              },
            },
          },
        },
      },
    },
  });
  return response.aggregations[agg1][agg2].buckets;
}

/**
 * Get edge for the given serial number.
 * @param serialNumber Serial number for the edge to get
 * @return Edge if found, null otherwise.
 */
export async function getEdgeBySerialNumber(
  serialNumber: string
): Promise<Edge> {
  // TODO FIXME - implement
  return null;
}

export async function getUserByEmail(email: string): Promise<User> {
  // TODO FIXME - implement
  return null;
}

/**
 * Convenience function to call esClient.search(params), and
 * (1) ensure params.version = true
 * (2) convert ES objects in result to model objects using modelFromEs.
 *
 * @param params
 * @returns {Promise<{id: string; version: number}[]>}
 */
export async function searchObjects<T>(params): Promise<T[]> {
  const params_ = params.version ? params : { version: true, ...params };
  const result = await esClient.search(params_);
  return result.hits.hits.map(obj => modelFromEs(obj));
}

// helper function to search for docs of the given type
// Note: this is for basic debugging and result set has the default max size of 10
export async function getAllDocs<T>(
  tenantId: string,
  type: DocType
): Promise<T[]> {
  return searchObjects<T>({
    type,
    index: tenantId,
    version: true,
  });
}

/**
 * Delete the given ElasticSearch index if exists.
 * @param {string} index
 * @returns {Promise<any>}
 */
export async function deleteIndex(index: string): Promise<any> {
  try {
    const idx_exists = await esClient.indices.exists({ index });
    if (idx_exists) {
      return esClient.indices.delete({
        index,
      });
    }
  } catch (e) {
    // ignore
  }
  return Promise.resolve();
}

export async function createTenant(
  index: string,
  body
): Promise<CreateDocumentResponse> {
  const tenantId = body.id;
  const doc = omit(body, ['id']);

  // create tenant with the given id
  const esClient = getESClient();
  const ur = await esClient.update({
    index,
    type: DocType.Tenant,
    id: tenantId,
    body: {
      doc,
      doc_as_upsert: true,
    },
    refresh: true,
  });

  // set up tenant alias
  await esClient.indices.putAlias({
    index,
    name: tenantId,
    body: {
      routing: tenantId,
      filter: {
        term: {
          tenantId,
        },
      },
    },
  });
  return { _id: tenantId };
}

export async function deleteExistingTenants(index: string) {
  // see if tenant with same name exists
  const tenants: Tenant[] = await searchObjects<Tenant>({
    index,
    type: DocType.Tenant,
    version: true,
  });
  if (tenants.length) {
    logger.info('>>> deleting ' + tenants.length + ' tenants...');
    const pms = tenants.map(tnt =>
      esClient.delete({
        index,
        id: tnt.id,
        type: DocType.Tenant,
      })
    );
    await Promise.all(pms).then(() => {
      // refresh
      return esClient.indices.refresh({ index });
    });
  }
}

async function createIndexMaybe(index) {
  // create index if does not exist
  try {
    const idx_exists = await esClient.indices.exists({ index });
    if (!idx_exists) {
      return esClient.indices.create({
        index,
      });
    }
  } catch (e) {
    // ignore
  }
  return Promise.resolve();
}

export function refreshIndex(index): Promise<any> {
  return esClient.indices.refresh({ index });
}

export async function createAllTables(): Promise<void> {
  // no op
  return;
}

export async function deleteAllTables(): Promise<void> {
  // no op
  return;
}
