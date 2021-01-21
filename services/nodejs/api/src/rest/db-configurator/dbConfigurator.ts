import {
  UpdateDocumentResponse,
  CreateDocumentResponse,
  DeleteDocumentResponse,
} from '../model/baseModel';

import { createDocument as esCreateDocument } from '../es-api/baseApi';
import { deleteDocument as esDeleteDocument } from '../es-api/baseApi';
import { updateDocument as esUpdateDocument } from '../es-api/baseApi';
import { getDocument as esGetDocument } from '../es-api/baseApi';
import { getAllDocuments as esGetAllDocuments } from '../es-api/baseApi';
import { getAllDocumentsForEdge as esGetAllDocumentsForEdge } from '../es-api/baseApi';
import { getEdgeBySerialNumber as esGetEdgeBySerialNumber } from '../es-api/baseApi';
import { getAggregate as esGetAggregate } from '../es-api/baseApi';
import { getNestedAggregate as esGetNestedAggregate } from '../es-api/baseApi';
import { createTenant as esCreateTenant } from '../es-api/baseApi';
import { createAllTables as esCreateAllTables } from '../es-api/baseApi';
import { deleteAllTables as esDeleteAllTables } from '../es-api/baseApi';
import { getUserByEmail as esGetUserByEmail } from '../es-api/baseApi';

import { createDocument as sqlCreateDocument } from '../sql-api/baseApi';
import { deleteDocument as sqlDeleteDocument } from '../sql-api/baseApi';
import { updateDocument as sqlUpdateDocument } from '../sql-api/baseApi';
import { getDocument as sqlGetDocument } from '../sql-api/baseApi';
import { findOneDocument as sqlFindOneDocument } from '../sql-api/baseApi';
import { getAllDocuments as sqlGetAllDocuments } from '../sql-api/baseApi';
import { getAllDocumentsForEdge as sqlGetAllDocumentsForEdge } from '../sql-api/baseApi';
import { getEdgeBySerialNumber as sqlGetEdgeBySerialNumber } from '../sql-api/baseApi';
import { getAggregate as sqlGetAggregate } from '../sql-api/baseApi';
import { getNestedAggregate as sqlGetNestedAggregate } from '../sql-api/baseApi';
import { createTenant as sqlCreateTenant } from '../sql-api/baseApi';
import { createAllTables as sqlCreateAllTables } from '../sql-api/baseApi';
import { deleteAllTables as sqlDeleteAllTables } from '../sql-api/baseApi';
import { getUserByEmail as sqlGetUserByEmail } from '../sql-api/baseApi';
import { getEdgeCert as sqlGetEdgeCert } from '../sql-api/baseApi';
import { updateEdgeCert as sqlUpdateEdgeCert } from '../sql-api/baseApi';
import { createEdgeCert as sqlCreateEdgeCert } from '../sql-api/baseApi';

import { Edge, User, EdgeCert } from '../model/index';
import { DocType } from '../model/baseModel';
import { String } from 'aws-sdk/clients/cognitosync';

// The purpose of DB configurator is to allow us
// to easily switch between DB service implementations
// at runtime.
// DB service specific implementation are in
//   es-api/baseApi - for ElasticSearch
//   sql-api/baseApi - for SQL
// the above should only be accessed directly from here.
// The rest of the app should use getDBService() here
// to access the above via DBService interface.

// SQL | ELASTICSEARCH
const DB_SERVICE_SQL = 'SQL';
const DB_SERVICE_ELASTICSEARCH = 'ELASTICSEARCH';
const DB_SERVICE: string =
  process.env.SHERLOCK_MGMT_DB_SERVICE || DB_SERVICE_SQL;

export function isElasticSearch() {
  return DB_SERVICE === DB_SERVICE_ELASTICSEARCH;
}
export function isSQL() {
  return DB_SERVICE === DB_SERVICE_SQL;
}

function notImplemented(...args): Promise<any> {
  throw Error('Not Implemented');
}

export interface DBService {
  createDocument(
    tenantId: string,
    type: DocType,
    doc
  ): Promise<CreateDocumentResponse>;

  deleteDocument(
    tenantId: string,
    id: string,
    type: DocType
  ): Promise<DeleteDocumentResponse>;

  updateDocument(
    tenantId: string,
    id: string,
    type: DocType,
    doc
  ): Promise<UpdateDocumentResponse>;

  getDocument<T>(tenantId: string, id: string, type: DocType): Promise<T>;

  findOneDocument<T>(tenantId: string, where: any, type: DocType): Promise<T>;

  getAllDocuments<T>(tenantId: string, type: DocType): Promise<T[]>;

  getAllDocumentsForEdge<T>(
    tenantId: string,
    edgeId: string,
    type: DocType
  ): Promise<T[]>;

  getEdgeBySerialNumber(serialNumber: string): Promise<Edge>;

  getUserByEmail(email: string): Promise<User>;

  getAggregate(
    tenantId: string,
    type: DocType,
    fieldName: string
  ): Promise<any>;

  getNestedAggregate(
    tenantId: string,
    type: DocType,
    fieldName: string,
    nestedFieldName: string
  ): Promise<any>;

  createTenant(index: string, body): Promise<CreateDocumentResponse>;

  createAllTables(): Promise<void>;

  deleteAllTables(): Promise<void>;

  getEdgeCert(edgeId: String): Promise<EdgeCert>;

  updateEdgeCert(cert: EdgeCert): Promise<void>;

  createEdgeCert(tenantId: string, edgeId: string): Promise<void>;
}

const DB_IMPL_ELASTICSEARCH: DBService = {
  createDocument: esCreateDocument,
  deleteDocument: esDeleteDocument,
  updateDocument: esUpdateDocument,
  getDocument: esGetDocument,
  findOneDocument: notImplemented,
  getAllDocuments: esGetAllDocuments,
  getAllDocumentsForEdge: esGetAllDocumentsForEdge,
  getEdgeBySerialNumber: esGetEdgeBySerialNumber,
  getAggregate: esGetAggregate,
  getNestedAggregate: esGetNestedAggregate,
  createTenant: esCreateTenant,
  createAllTables: esCreateAllTables,
  deleteAllTables: esDeleteAllTables,
  getUserByEmail: esGetUserByEmail,
  getEdgeCert: notImplemented,
  updateEdgeCert: notImplemented,
  createEdgeCert: notImplemented,
};

const DB_IMPL_SQL: DBService = {
  createDocument: sqlCreateDocument,
  deleteDocument: sqlDeleteDocument,
  updateDocument: sqlUpdateDocument,
  getDocument: sqlGetDocument,
  findOneDocument: sqlFindOneDocument,
  getAllDocuments: sqlGetAllDocuments,
  getAllDocumentsForEdge: sqlGetAllDocumentsForEdge,
  getEdgeBySerialNumber: sqlGetEdgeBySerialNumber,
  getAggregate: sqlGetAggregate,
  getNestedAggregate: sqlGetNestedAggregate,
  createTenant: sqlCreateTenant,
  createAllTables: sqlCreateAllTables,
  deleteAllTables: sqlDeleteAllTables,
  getUserByEmail: sqlGetUserByEmail,
  getEdgeCert: sqlGetEdgeCert,
  updateEdgeCert: sqlUpdateEdgeCert,
  createEdgeCert: sqlCreateEdgeCert,
};

export function getDBService(): DBService {
  switch (DB_SERVICE) {
    case DB_SERVICE_SQL:
      return DB_IMPL_SQL;
    case DB_SERVICE_ELASTICSEARCH:
      return DB_IMPL_ELASTICSEARCH;
    default:
      throw Error(`getDBService: unsupported DB ${DB_SERVICE}`);
  }
}
