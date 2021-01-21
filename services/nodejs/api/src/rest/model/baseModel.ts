export enum DocType {
  Tenant = 'tenant',
  Edge = 'edge',
  Script = 'script',
  Category = 'category',
  CategoryValue = 'categoryvalue',
  DataSource = 'datasource',
  DataSourceField = 'datasourcefield',
  DataSourceFieldSelector = 'datasourcefieldselector',
  DataStream = 'datastream',
  DataStreamOrigin = 'datastreamorigin',
  Sensor = 'sensor',
  User = 'user',
  Project = 'project',
  CloudCreds = 'cloudcreds',
  EdgeCert = 'edgecert',
  Log = 'log',
  Settings = 'settings',
  Application = 'application',
  ApplicationStatus = 'applicationstatus',
  DockerProfile = 'dockerprofile',
  ScriptRuntime = 'scriptruntime',
  ProjectUser = 'projectuser',
  ProjectDockerProfile = 'projectdockerprofile',
  ProjectCloudCreds = 'projectcloudcreds',
  EdgeInfo = 'edgeinfo',
  Domain = 'domain',
  TenantRootCA = 'tenantrootca',
}
export const DocTypes = [
  DocType.Tenant,
  DocType.Edge,
  DocType.Script,
  DocType.Category,
  DocType.CategoryValue,
  DocType.DataSource,
  DocType.DataSourceField,
  DocType.DataSourceFieldSelector,
  DocType.DataStream,
  DocType.DataStreamOrigin,
  DocType.Sensor,
  DocType.User,
  DocType.Project,
  DocType.CloudCreds,
  DocType.EdgeCert,
  DocType.Log,
  DocType.Application,
  DocType.ApplicationStatus,
  DocType.DockerProfile,
  DocType.ScriptRuntime,
  DocType.ProjectUser,
  DocType.ProjectDockerProfile,
  DocType.ProjectCloudCreds,
  DocType.EdgeInfo,
  DocType.Domain,
  DocType.TenantRootCA,
];

/**
 * BaseModel
 * All objects except for Tenant should extend this.
 * (E.g., DataStreams, Scripts)
 */
export interface BaseModel {
  /**
   * Unique id to identify the object.
   * id is marked optional as it is not required during create.
   * id could be supplied during create or DB generated.
   */
  id?: string;
  /**
   * Version number of object maintained by DB.
   * Not currently used.
   * version is marked optional as it is not required during create/update.
   */
  version?: number;
  /**
   * Id of tenant this object belongs to.
   */
  tenantId: string;
}
export const BaseModelKeys = ['id', 'version', 'tenantId'];

/**
 * EdgeBaseModel
 * All objects belonging to an Edge should extend this.
 * (e.g., Sensors, DataSources)
 */
export interface EdgeBaseModel extends BaseModel {
  /**
   * Id of edge this object belongs to.
   */
  edgeId: string;
}
export const EdgeBaseModelKeys = ['edgeId'].concat(BaseModelKeys);

/**
 * Spec used for aggregate request
 */
export interface AggregateSpec {
  /**
   * entity type to perform aggregate query
   */
  type: DocType;
  /**
   * field of the entity to perform aggregate query
   */
  field: string;
}
export const AggregateSpecKeys = ['type', 'field'];

/**
 * Spec used for nested aggregate request
 */
export interface NestedAggregateSpec {
  /**
   * entity type to perform nested aggregate query
   */
  type: DocType;
  /**
   * field of the entity to perform nested aggregate query
   */
  field: string;
  /**
   * nested field of the entity field to perform nested aggregate query
   */
  nestedField: string;
}
export const NestedAggregateSpecKeys = ['type', 'field', 'nestedField'];

/**
 * Aggregate query response format
 */
export interface AggregateInfo {
  key: string;
  doc_count: number;
}
export const AggregateInfoKeys = ['key', 'doc_count'];

/**
 * Spec for a script parameter
 */
export interface ScriptParam {
  /**
   * Name of the parameter
   */
  name: string;
  /**
   * Type of the parameter
   */
  type: string;
}
/**
 * Instance of a script parameter value
 */
export interface ScriptParamValue extends ScriptParam {
  /**
   * Value of the parameter
   */
  value: number | string;
}
/**
 * Transformation ID and args info for use of
 * transformation in DataStream.
 */
export interface TransformationArgs {
  /**
   * ID for the transformation
   */
  transformationId: string;
  /**
   * Array of script param values for the transformation
   */
  args: ScriptParamValue[];
}

// ElasticSearch update document response
// export interface UpdateDocumentResponse {
//   _shards: ShardsResponse;
//   _index: string;
//   _type: string;
//   _id: string;
//   _version: number;
//   result: string;
//   forced_refresh: boolean;
// }
/**
 * @tsoaModel
 */
export interface UpdateDocumentResponse {
  _id: string;
  result?: string;
}

export const RESULT_NOOP = 'noop';
export const RESULT_UPDATE_SUCCESS = 'updated';
export const RESULT_CREATE_SUCCESS = 'created';
export const RESULT_DELETE_SUCCESS = 'deleted';

// dupe from elasticsearch to work around tsoa issue {
// export interface CreateDocumentResponse {
//   _shards: ShardsResponse;
//   _index: string;
//   _type: string;
//   _id: string;
//   _version: number;
//   created: boolean;
//   result: string;
// }
/**
 * @tsoaModel
 */
export interface CreateDocumentResponse {
  _id: string;
  result?: string;
  // TODO FIXME - add status, also use try/catch with await
}
// export interface DeleteDocumentResponse {
//   _shards: ShardsResponse;
//   found: boolean;
//   _index: string;
//   _type: string;
//   _id: string;
//   _version: number;
//   result: string;
// }
/**
 * @tsoaModel
 */
export interface DeleteDocumentResponse {
  _id: string;
  result?: string;
}
export interface ShardsResponse {
  total: number;
  successful: number;
  failed: number;
}
// dupe from elasticsearch to work around tsoa issue }

export const DATA_TYPES = [
  'All',
  'Custom',
  'Humidity',
  'Image',
  'Light',
  'Motion',
  'Pressure',
  'Processed',
  'Proximity',
  'Temperature',
];

export enum CloudType {
  SELECT = 'select',
  AWS = 'AWS',
  GCP = 'GCP',
}

export interface Exception extends Error {
  status: number;
}
