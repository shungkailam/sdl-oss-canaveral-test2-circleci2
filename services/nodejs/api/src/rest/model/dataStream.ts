import {
  BaseModel,
  BaseModelKeys,
  CloudType,
  TransformationArgs,
} from './baseModel';
import { CategoryInfo } from './category';

export enum AWS_REGION {
  US_EAST_2 = 'us-east-2',
  US_EAST_1 = 'us-east-1',
  US_WEST_1 = 'us-west-1',
  US_WEST_2 = 'us-west-2',
  AP_NORTHEAST_1 = 'ap-northeast-1',
  AP_NORTHEAST_2 = 'ap-northeast-2',
  AP_NORTHEAST_3 = 'ap-northeast-3',
  AP_SOUTH_1 = 'ap-south-1',
  AP_SOUTHEAST_1 = 'ap-southeast-1',
  AP_SOUTHEAST_2 = 'ap-southeast-2',
  CA_CENTRAL_1 = 'ca-central-1',
  CN_NORTH_1 = 'cn-north-1',
  CN_NORTHWEST_1 = 'cn-northwest-1',
  EU_CENTRAL_1 = 'eu-central-1',
  EU_WEST_1 = 'eu-west-1',
  EU_WEST_2 = 'eu-west-2',
  EU_WEST_3 = 'eu-west-3',
  EU_EAST_1 = 'sa-east-1',
}
export const AWS_REGIONS: AWS_REGION[] = [
  AWS_REGION.US_EAST_1,
  AWS_REGION.US_EAST_2,
  AWS_REGION.US_WEST_1,
  AWS_REGION.US_WEST_2,
  AWS_REGION.AP_NORTHEAST_1,
  AWS_REGION.AP_NORTHEAST_2,
  AWS_REGION.AP_NORTHEAST_3,
  AWS_REGION.AP_SOUTH_1,
  AWS_REGION.AP_SOUTHEAST_1,
  AWS_REGION.AP_SOUTHEAST_2,
  AWS_REGION.CA_CENTRAL_1,
  AWS_REGION.CN_NORTH_1,
  AWS_REGION.CN_NORTHWEST_1,
  AWS_REGION.EU_CENTRAL_1,
  AWS_REGION.EU_WEST_1,
  AWS_REGION.EU_WEST_2,
  AWS_REGION.EU_WEST_3,
  AWS_REGION.EU_EAST_1,
];
export enum GCP_REGION {
  NA_NORTHEAST1 = 'northamerica-northeast1',
  US_CENTRAL1 = 'us-central1',
  US_WEST1 = 'us-west1',
  US_EAST4 = 'us-east4',
  US_EAST1 = 'us-east1',
  SA_EAST1 = 'southamerica-east1',
  EU_WEST1 = 'europe-west1',
  EU_WEST2 = 'europe-west2',
  EU_WEST3 = 'europe-west3',
  EU_WEST4 = 'europe-west4',
  ASIA_SOUTH1 = 'asia-south1',
  ASIA_SOUTHEAST1 = 'asia-southeast1',
  ASIA_EAST1 = 'asia-east1',
  ASIA_NORTHEAST1 = 'asia-northeast1',
  AS_SOUTHEAST1 = 'australia-southeast1',
}
export const GCP_REGIONS: GCP_REGION[] = [
  GCP_REGION.NA_NORTHEAST1,
  GCP_REGION.US_CENTRAL1,
  GCP_REGION.US_WEST1,
  GCP_REGION.US_EAST1,
  GCP_REGION.US_EAST4,
  GCP_REGION.SA_EAST1,
  GCP_REGION.EU_WEST1,
  GCP_REGION.EU_WEST2,
  GCP_REGION.EU_WEST3,
  GCP_REGION.EU_WEST4,
  GCP_REGION.ASIA_SOUTH1,
  GCP_REGION.ASIA_SOUTHEAST1,
  GCP_REGION.ASIA_EAST1,
  GCP_REGION.ASIA_NORTHEAST1,
  GCP_REGION.AS_SOUTHEAST1,
];
export enum DataStreamDestination {
  Edge = 'Edge',
  Cloud = 'Cloud',
}
export const DataStreamDestinations: DataStreamDestination[] = [
  DataStreamDestination.Edge,
  DataStreamDestination.Cloud,
];
export enum EdgeStreamType {
  Kafka = 'Kafka',
  ElasticSearch = 'ElasticSearch',
  MQTT = 'MQTT',
  None = 'None', // NATS only
}
export const EdgeStreamTypes: EdgeStreamType[] = [
  EdgeStreamType.Kafka,
  EdgeStreamType.ElasticSearch,
  EdgeStreamType.MQTT,
  EdgeStreamType.None,
];
export enum AWSStreamType {
  Kinesis = 'Kinesis',
  SQS = 'SQS',
  S3 = 'S3',
  DynamoDB = 'DynamoDB',
}
export const AWSStreamTypes: AWSStreamType[] = [
  AWSStreamType.Kinesis,
  AWSStreamType.SQS,
  AWSStreamType.S3,
  AWSStreamType.DynamoDB,
];
export enum GCPStreamType {
  PubSub = 'PubSub',
  CloudDatastore = 'CloudDatastore',
  CloudSQL = 'CloudSQL',
}
export const GCPStreamTypes: GCPStreamType[] = [
  GCPStreamType.PubSub,
  GCPStreamType.CloudDatastore,
  GCPStreamType.CloudSQL,
];
export interface RetentionInfo {
  /**
   * Retention type can be Time or Size.
   * For Time based retention, limit is in seconds
   * and specify how long the data should be retained.
   * For Size based retention, limit is in GB
   * and specify up to what capacity the data should
   * be retained.
   */
  type: 'Time' | 'Size';
  limit: number;
}
export const RetentionInfoKeys = ['type', 'limit'];

/**
 * DataStreams are fundamental building blocks for Sherlock data pipeline.
 */
export interface DataStream extends BaseModel {
  /**
   * Name of the DataStream.
   * This is the published output (Kafka topic) name.
   */
  name: string;
  /**
   * name for the DataStream
   */
  description?: string;
  /**
   * Data type of the DataStream.
   * E.g., Temperature, Pressure, Image, Multiple, etc.
   */
  dataType: string;
  /**
   * The origin of the DataStream.
   * Either 'Data Source' or 'Data Stream'
   */
  origin: 'Data Source' | 'Data Stream';
  /**
   * A list of CategoryInfo used as criteria
   * to filter sources applicable to this DataStream.
   */
  originSelectors: CategoryInfo[];
  /**
   * If origin == 'Data Stream', then originId
   * can be used in place of originSelectors
   * to specify the origin data stream id if the origin data stream is unique.
   */
  originId?: string;
  /**
   * Destination of the DataStream.
   * Either Edge or Cloud.
   */
  destination: DataStreamDestination;
  /**
   * Cloud type, required if destination == Cloud
   */
  cloudType?: CloudType;
  /**
   * CloudCreds id.
   * Required if destination == Cloud
   */
  cloudCredsId?: string;
  /**
   * AWS region - required if cloudType == AWS
   */
  awsCloudRegion?: AWS_REGION;
  /**
   * GCP region - required if cloudType == GCP
   */
  gcpCloudRegion?: GCP_REGION;
  /**
   * Type of the DataStream at Edge.
   * Required if destination == Edge
   */
  edgeStreamType?: EdgeStreamType;
  /**
   * Type of the DataStream at AWS Cloud.
   * Required if cloudType == AWS
   */
  awsStreamType?: AWSStreamType;
  /**
   * Type of the DataStream at GCP Cloud.
   * Required if cloudType == GCP
   */
  gcpStreamType?: GCPStreamType;
  /**
   * Current size of the DataStream output in GB.
   */
  size: number;

  /**
   * Whether to turn sampling on.
   * If true, then samplingInterval should be set as well.
   */
  enableSampling: boolean;
  /**
   * Sampling interval in seconds.
   * The sampling interval applies to each mqtt/kafka topic separately.
   */
  samplingInterval?: number;

  /**
   * List of transformations (together with their args)
   * to apply to the origin data
   * to produce the destination data.
   * Could be null or empty if no transformation required.
   * Each entry is the id of the transformation Script to apply to input from origin
   * to produce output to destination.
   */
  transformationArgsList: TransformationArgs[];

  /**
   * Retention policy for this DataStream.
   * Multiple RetentionInfo are combined using AND semantics.
   * E.g., retain data for 1 month AND up to 2 TB.
   */
  dataRetention: RetentionInfo[];

  projectId: string;

  endPoint: string;
}
export const DataStreamKeys = [
  'name',
  'dataType',
  'origin',
  'originSelectors',
  'originId',
  'destination',
  'cloudType',
  'cloudCredsId',
  'awsCloudRegion',
  'gcpCloudRegion',
  'edgeStreamType',
  'awsStreamType',
  'gcpStreamType',
  'size',
  'enableSampling',
  'samplingInterval',
  'transformationArgsList',
  'dataRetention',
  'endPoint',
].concat(BaseModelKeys);
