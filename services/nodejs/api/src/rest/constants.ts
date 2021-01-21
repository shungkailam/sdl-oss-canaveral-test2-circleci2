export const GLOBAL_INDEX_NAME = 'mgmt';

export enum DynamoTableName {
  Tenant = 'Tenant',
  Edge = 'Edge',
  Script = 'Script',
  Category = 'Category',
  DataSource = 'DataSource',
  DataStream = 'DataStream',
  Sensor = 'Sensor',
  User = 'User',
  Project = 'Project',
  CloudCreds = 'CloudCreds',
}

export const DynamoTableNames = [
  DynamoTableName.Tenant,
  DynamoTableName.Edge,
  DynamoTableName.Script,
  DynamoTableName.Category,
  DynamoTableName.DataSource,
  DynamoTableName.DataStream,
  DynamoTableName.Sensor,
  DynamoTableName.User,
  DynamoTableName.Project,
  DynamoTableName.CloudCreds,
];
// Kafka topic name for each object type.
// Note: this info is stored in common/kafkatopics.json
export const KAFKA_NOTIFICATION_TOPIC_CATEGORY = 'categoryevents';
export const KAFKA_NOTIFICATION_TOPIC_DATASOURCE = 'datasourceevents';
export const KAFKA_NOTIFICATION_TOPIC_DATASTREAM = 'datastreamevents';
export const KAFKA_NOTIFICATION_TOPIC_SCRIPT = 'scriptevents';
export const KAFKA_NOTIFICATION_TOPIC_SENSOR = 'sensorevents';
