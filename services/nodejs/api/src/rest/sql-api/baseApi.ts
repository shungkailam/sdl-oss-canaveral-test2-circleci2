import { Sequelize } from 'sequelize-typescript';
import {
  DocType,
  DocTypes,
  UpdateDocumentResponse,
  CreateDocumentResponse,
  DeleteDocumentResponse,
  Exception,
} from '../model/baseModel';

import { Edge, User, DataStream, EdgeCert, Tenant } from '../model/index';

import { CategoryModel } from '../sql-model/CategoryModel';
import { CategoryValueModel } from '../sql-model/CategoryValueModel';
import { DataSourceModel } from '../sql-model/DataSourceModel';
import { DataSourceFieldModel } from '../sql-model/DataSourceFieldModel';
import { DataSourceFieldSelectorModel } from '../sql-model/DataSourceFieldSelectorModel';
import { DataStreamModel } from '../sql-model/DataStreamModel';
import { DataStreamOriginModel } from '../sql-model/DataStreamOriginModel';
import { EdgeModel } from '../sql-model/EdgeModel';
import { ScriptModel } from '../sql-model/ScriptModel';
import { SensorModel } from '../sql-model/SensorModel';
import { TenantModel } from '../sql-model/TenantModel';
import { UserModel } from '../sql-model/UserModel';
import { ProjectModel } from '../sql-model/ProjectModel';
import { CloudCredsModel } from '../sql-model/CloudCredsModel';
import { EdgeCertModel } from '../sql-model/EdgeCertModel';
import { LogModel } from '../sql-model/LogModel';
import { ApplicationModel } from '../sql-model/ApplicationModel';
import { ApplicationStatusModel } from '../sql-model/ApplicationStatusModel';
import { DockerProfileModel } from '../sql-model/DockerProfileModel';
import { ScriptRuntimeModel } from '../sql-model/ScriptRuntimeModel';
import { ProjectUserModel } from '../sql-model/ProjectUserModel';
import { ProjectDockerProfileModel } from '../sql-model/ProjectDockerProfileModel';
import { ProjectCloudCredsModel } from '../sql-model/ProjectCloudCredsModel';
import { EdgeInfoModel } from '../sql-model/EdgeInfoModel';
import { DomainModel } from '../sql-model/DomainModel';
import { TenantRootCAModel } from '../sql-model/TenantRootCAModel';

import { logger } from '../util/logger';
import * as uuidv4 from 'uuid/v4';
import * as omit from 'object.omit';
import { getCerts, CertData } from '../../getCerts/getCerts';
import platformService from '../services/platform.service';

const DEBUG = false;

export async function ignorePromiseError(promise) {
  try {
    return await promise;
  } catch (e) {
    logger.info('ignore promise error: ', e);
    return null;
  }
}

export function getModel(type: DocType): any {
  switch (type) {
    case DocType.Category:
      return CategoryModel;
    case DocType.CategoryValue:
      return CategoryValueModel;
    case DocType.DataSource:
      return DataSourceModel;
    case DocType.DataSourceField:
      return DataSourceFieldModel;
    case DocType.DataSourceFieldSelector:
      return DataSourceFieldSelectorModel;
    case DocType.DataStream:
      return DataStreamModel;
    case DocType.DataStreamOrigin:
      return DataStreamOriginModel;
    case DocType.Edge:
      return EdgeModel;
    case DocType.Script:
      return ScriptModel;
    case DocType.Sensor:
      return SensorModel;
    case DocType.Tenant:
      return TenantModel;
    case DocType.User:
      return UserModel;
    case DocType.Project:
      return ProjectModel;
    case DocType.CloudCreds:
      return CloudCredsModel;
    case DocType.EdgeCert:
      return EdgeCertModel;
    case DocType.Log:
      return LogModel;
    case DocType.Application:
      return ApplicationModel;
    case DocType.ApplicationStatus:
      return ApplicationStatusModel;
    case DocType.DockerProfile:
      return DockerProfileModel;
    case DocType.ScriptRuntime:
      return ScriptRuntimeModel;
    case DocType.ProjectUser:
      return ProjectUserModel;
    case DocType.ProjectDockerProfile:
      return ProjectDockerProfileModel;
    case DocType.ProjectCloudCreds:
      return ProjectCloudCredsModel;
    case DocType.EdgeInfo:
      return EdgeInfoModel;
    case DocType.Domain:
      return DomainModel;
    case DocType.TenantRootCA:
      return TenantRootCAModel;
    default:
      break;
  }
  throw Error(`Model not found for type ${type}`);
}

class InvalidDataError implements Exception {
  public status = 400;
  public name = 'InvalidDataError';
  constructor(public message: string) {}
}

function modelFromSQL(type: DocType, data) {
  // id is required
  logger.debug('modelFromSQL {, data=', data);
  if (!data || !data.id) {
    logger.warn('modelFromSQL: bad input data:', data);
    throw new InvalidDataError('Bad model data ' + data);
  }
  // special handling of DataType.BOOLEAN:
  // convert number to boolean
  if (type === DocType.Edge) {
    const edge: Edge = data as Edge;
    edge.connected = !!edge.connected;
  } else if (type === DocType.DataStream) {
    const ds: DataStream = data as DataStream;
    ds.enableSampling = !!ds.enableSampling;
  }
  return data;
}

function isNoID(type) {
  return (
    type === DocType.CategoryValue ||
    type === DocType.DataStreamOrigin ||
    type === DocType.DataSourceField ||
    type === DocType.DataSourceFieldSelector ||
    type === DocType.ApplicationStatus ||
    type === DocType.ProjectUser ||
    type === DocType.ProjectDockerProfile ||
    type === DocType.ProjectCloudCreds
  );
}

/**
 * Create a document with the given type for the tenant.
 * If body.id exists, will use update with upsert,
 * else will use index call to create.
 * @param tenantId
 * @param type
 * @param doc {T} where T is the model for type
 * @returns {Promise<CreateDynamoDocumentResponse>}
 */
export function createDocument(
  tenantId: string,
  type: DocType,
  doc
): Promise<CreateDocumentResponse> {
  const st = Date.now();
  return new Promise<CreateDocumentResponse>(async (resolve, reject) => {
    try {
      const model = getModel(type);
      if (!doc.id && !isNoID(type)) {
        doc.id = uuidv4();
      }
      await model.upsert(doc);
      if (DEBUG) {
        logger.info(
          `createDocument, tenantId=${tenantId}, type=${type}, id=${
            doc.id
          } done in ${Date.now() - st}ms`
        );
      }
      resolve({ _id: doc.id });
    } catch (e) {
      logger.warn(
        `createDocument error, tenantId=${tenantId}, type=${type}, id=${doc.id}`
      );
      reject(e);
    }
  });
}

/**
 * Delete the document with the given id, type for the tenant.
 * Will succeed if doc with id does not exist
 * @param tenantId
 * @param id
 * @param type
 * @returns {Promise<DeleteDynamoDocumentResponse>}
 */
export function deleteDocument(
  tenantId: string,
  id: string,
  type: DocType
): Promise<DeleteDocumentResponse> {
  const st = Date.now();
  return new Promise(async (resolve, reject) => {
    try {
      const model = getModel(type);
      await model.destroy({
        where: { id },
      });
      if (DEBUG) {
        logger.info(
          `deleteDocument, tenantId=${tenantId}, type=${type}, id=${id}, done in ${Date.now() -
            st}ms`
        );
      }
      resolve({ _id: id });
    } catch (e) {
      logger.warn(
        `deleteDocument error, tenantId=${tenantId}, type=${type}, id=${id}`
      );
      reject(e);
    }
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
  return findOneDocument<T>(tenantId, { id }, type);
}

export async function findOneDocument<T>(
  tenantId: string,
  where: any,
  type: DocType
): Promise<T> {
  const st = Date.now();
  return new Promise<T>(async (resolve, reject) => {
    try {
      const model = getModel(type);
      let doc = await model.findOne({ where, raw: true });
      if (DEBUG) {
        logger.info(
          `findOneDocument, tenantId=${tenantId}, type=${type}, done in ${Date.now() -
            st}ms`,
          where
        );
      }
      if (doc) {
        doc = modelFromSQL(type, doc);
      }
      resolve(doc);
    } catch (e) {
      logger.warn(
        `getDocument error, tenantId=${tenantId}, type=${type},`,
        where
      );
      reject(e);
    }
  });
}

/**
 * Update the document with the given id, type for the tenant.
 * @param tenantId
 * @param id
 * @param type
 * @param doc
 * @returns {Promise<UpdateDynamoDocumentResponse>}
 * @throws Will throw if not found
 */
export async function updateDocument(
  tenantId: string,
  id: string,
  type: DocType,
  doc
): Promise<UpdateDocumentResponse> {
  const st = Date.now();
  return new Promise<UpdateDocumentResponse>(async (resolve, reject) => {
    try {
      const rest = omit(doc, ['id']);
      const model = getModel(type);
      await model.update(rest, { where: { id } });
      if (DEBUG) {
        logger.info(
          `updateDocument, tenantId=${tenantId}, type=${type}, id=${
            doc.id
          }, done in ${Date.now() - st}ms`
        );
      }
      resolve({ _id: id });
    } catch (e) {
      logger.warn(
        `updateDocument error, tenantId=${tenantId}, type=${type}, id=${doc.id}`
      );
      reject(e);
    }
  });
}

export async function getAllDocuments<T>(
  tenantId: string,
  type: DocType
): Promise<T[]> {
  const st = Date.now();
  return new Promise<T[]>(async (resolve, reject) => {
    try {
      const model = getModel(type);
      let docs: T[] = [];
      if (type === DocType.Tenant || type === DocType.TenantRootCA) {
        docs = await model.findAll({ raw: true });
      } else {
        docs = await model.findAll({ where: { tenantId }, raw: true });
      }
      if (DEBUG) {
        logger.info(
          `getAllDocuments, tenantId=${tenantId}, type=${type}, done in ${Date.now() -
            st}ms`
        );
      }
      resolve(docs.map(doc => modelFromSQL(type, doc)));
    } catch (e) {
      logger.warn(`getAllDocument error, tenantId=${tenantId}, type=${type}`);
      reject(e);
    }
  });
}

export async function getAllDocumentsForEdge<T>(
  tenantId: string,
  edgeId: string,
  type: DocType
): Promise<T[]> {
  const st = Date.now();
  return new Promise<T[]>(async (resolve, reject) => {
    try {
      const model = getModel(type);
      const docs = await model.findAll({
        where: { tenantId, edgeId },
        raw: true,
      });
      if (DEBUG) {
        logger.info(
          `getAllDocumentsForEdge, tenantId=${tenantId}, type=${type}, edgeId=${edgeId}, done in ${Date.now() -
            st}ms`
        );
      }
      resolve(docs.map(doc => modelFromSQL(type, doc)));
    } catch (e) {
      logger.warn(
        `getAllDocumentsForEdge error, tenantId=${tenantId}, type=${type}, edgeId=${edgeId}`
      );
      reject(e);
    }
  });
}

/**
 * Get edge for the given serial number.
 * @param serialNumber Serial number for the edge to get
 * @return Edge if found, null otherwise.
 */
export async function getEdgeBySerialNumber(
  serialNumber: string
): Promise<Edge> {
  return new Promise<Edge>(async (resolve, reject) => {
    try {
      const model = getModel(DocType.Edge);
      let doc = await model.findOne({
        where: { serialNumber },
        raw: true,
      });
      if (doc) {
        doc = modelFromSQL(DocType.Edge, doc);
      }
      resolve(doc);
    } catch (e) {
      logger.warn(`getEdgeBySerialNumber error, serialNumber=${serialNumber}`);
      reject(e);
    }
  });
}

export async function getUserByEmail(email: string): Promise<User> {
  return new Promise<User>(async (resolve, reject) => {
    try {
      const model = getModel(DocType.User);
      let doc = await model.findOne({
        where: { email },
        raw: true,
      });
      if (doc) {
        doc = modelFromSQL(DocType.User, doc);
      }
      resolve(doc);
    } catch (e) {
      logger.warn(`getUserByEmail error, email=${email}`);
      reject(e);
    }
  });
}

export async function getAggregate(
  tenantId: string,
  type: DocType,
  fieldName: string
): Promise<any> {
  const st = Date.now();
  return new Promise<any>(async (resolve, reject) => {
    try {
      const sequelize = initSequelize();
      const model = getModel(type);
      const alias = 'ids';
      const docs = (await model.findAll({
        where: { tenantId },
        group: [fieldName],
        attributes: [
          fieldName,
          [sequelize.fn('COUNT', sequelize.col('id')), alias],
        ],
        raw: true,
      })).map(e => ({ key: e[fieldName], doc_count: e[alias] }));
      if (DEBUG) {
        logger.info(
          `getAggregate, tenantId=${tenantId}, type=${type}, done in ${Date.now() -
            st}ms`
        );
      }
      resolve(docs);
    } catch (e) {
      logger.warn(
        `getAggregate error, tenantId=${tenantId}, type=${type}, field=${fieldName}`
      );
      reject(e);
    }
  });
}

export async function getNestedAggregate(
  tenantId: string,
  type: DocType,
  fieldName: string,
  nestedFieldName: string
): Promise<any> {
  const st = Date.now();
  return new Promise<any>(async (resolve, reject) => {
    try {
      const model = getModel(type);
      const docMap = (await model.findAll({
        where: { tenantId },
        attributes: [fieldName],
        raw: true,
      })).reduce((acc: any, cur) => {
        cur[fieldName].forEach(s => {
          const nfv = s[nestedFieldName];
          if (acc[nfv]) {
            acc[nfv] += 1;
          } else {
            acc[nfv] = 1;
          }
        });
        return acc;
      }, {});
      const docs = Object.keys(docMap).map(k => ({
        key: k,
        doc_count: docMap[k],
      }));
      if (DEBUG) {
        logger.info(
          `getNestedAggregate, tenantId=${tenantId}, type=${type}, done in ${Date.now() -
            st}ms`
        );
      }
      resolve(docs);
    } catch (e) {
      logger.warn(
        `getNestedAggregate error, tenantId=${tenantId}, type=${type}, field=${fieldName}, nested field=${nestedFieldName}`
      );
      reject(e);
    }
  });
}

export async function createTenant(
  index: string,
  body
): Promise<CreateDocumentResponse> {
  const tenantId = body.id;
  return createDocument(tenantId, DocType.Tenant, body);
}

export async function createTable(type: DocType): Promise<void> {
  return new Promise<void>(async (resolve, reject) => {
    try {
      const model = getModel(type);
      await model.sync({ force: false });
      resolve();
    } catch (e) {
      reject(e);
    }
  });
}

export async function deleteTable(type: DocType): Promise<void> {
  return new Promise<void>(async (resolve, reject) => {
    try {
      const model = getModel(type);
      await model.drop();
      resolve();
    } catch (e) {
      reject(e);
    }
  });
}

export async function createAllTables(): Promise<void> {
  await Promise.all(
    DocTypes.map(type => ignorePromiseError(createTable(type)))
  );
  return;
}

export async function deleteAllTables(): Promise<void> {
  await Promise.all(
    DocTypes.map(type => ignorePromiseError(deleteTable(type)))
  );
  logger.info('All tables deleted');
  return;
}

export async function getEdgeCert(edgeId: string): Promise<EdgeCert> {
  const model = getModel(DocType.EdgeCert);
  const doc: EdgeCert = await model.findOne({
    where: { edge_id: edgeId },
    raw: true,
  });
  return doc;
}

export async function updateEdgeCert(edgeCert: EdgeCert): Promise<void> {
  return new Promise<void>(async (resolve, reject) => {
    try {
      const { id, ...rest } = edgeCert;
      const model = getModel(DocType.EdgeCert);
      await model.update(rest, { where: { id } });
      resolve();
    } catch (e) {
      reject(e);
    }
  });
}
export async function createEdgeCert(
  tenantId: string,
  edgeId: string
): Promise<void> {
  return new Promise<void>(async (resolve, reject) => {
    try {
      const certData: CertData = await getCerts(tenantId, 'server');
      logger.info('createEdgeCert, certData:', certData);
      if (!certData) {
        reject(Error('Failed to create Cert for edge'));
      }
      const tenant = await getDocument<Tenant>(null, tenantId, DocType.Tenant);
      if (!tenant) {
        reject(`Tenant not found for ${tenantId}`);
        return;
      }
      // store encrypted private key
      const privateKey = await platformService
        .getKeyService()
        .tenantEncrypt(certData.PrivateKey, tenant.token);
      const certificate = certData.Certificate;
      const clientCertificate = certificate;
      const edgeCertificate = certificate;
      const clientPrivateKey = privateKey;
      const edgePrivateKey = privateKey;
      const edgeCert: EdgeCert = {
        tenantId,
        edgeId,
        certificate,
        clientCertificate,
        edgeCertificate,
        privateKey,
        clientPrivateKey,
        edgePrivateKey,
        id: uuidv4(),
        locked: false,
      };
      const model = getModel(DocType.EdgeCert);
      await model.create(edgeCert);
      resolve();
    } catch (e) {
      logger.error('createEdgeCert caught exception', e);
      reject(e);
    }
  });
}

const SQL_HOST =
  process.env.SQL_HOST ||
  'sherlock-pg-dev-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com';
const SQL_PORT = process.env.SQL_PORT || 5432;
const SQL_DB = process.env.SQL_DB || 'sherlock_test';
const SQL_DIALECT = process.env.SQL_DIALECT || 'postgres';
const SQL_USER = process.env.SQL_USER || 'root';
const SQL_PASSWORD = process.env.SQL_PASSWORD;

let gSequelize: any = null;
export function initSequelize() {
  if (!gSequelize) {
    gSequelize = new Sequelize(<any>{
      host: SQL_HOST,
      port: SQL_PORT,
      database: SQL_DB,
      dialect: SQL_DIALECT,
      username: SQL_USER,
      password: SQL_PASSWORD,
      modelPaths: [__dirname + '/../sql-model'],
      benchmark: true,
      multipleStatements: true,
      pool: {
        max: 60,
        min: 0,
        idle: 20000,
        acquire: 600000,
      },
      logging: (...args) => logger.debug.apply(logger, args),
    });
  }
  return gSequelize;
}
