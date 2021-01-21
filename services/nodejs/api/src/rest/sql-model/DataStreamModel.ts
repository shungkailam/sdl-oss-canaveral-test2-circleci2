import {
  Table,
  Column,
  Model,
  PrimaryKey,
  CreatedAt,
  UpdatedAt,
  DataType,
  AllowNull,
} from 'sequelize-typescript';

import { CategoryInfo } from '../model/category';
import {
  RetentionInfo,
  DataStreamDestination,
  CloudType,
  AWS_REGION,
  GCP_REGION,
  EdgeStreamType,
  AWSStreamType,
  GCPStreamType,
  TransformationArgs,
} from '../model/index';
import { getJsonType } from '../util/dbUtil';

const JSON_TYPE = getJsonType();

@Table({ timestamps: true, tableName: 'data_stream_model' })
export class DataStreamModel extends Model<DataStreamModel> {
  @PrimaryKey
  @Column
  id: string;

  @CreatedAt created_at: Date;

  @UpdatedAt updated_at: Date;

  @AllowNull(false)
  @Column
  version: number;

  @AllowNull(false)
  @Column({ field: 'tenant_id' })
  tenantId: string;

  @AllowNull(false)
  @Column
  name: string;

  @AllowNull(true)
  @Column
  description: string;

  @AllowNull(false)
  @Column({ field: 'data_type' })
  dataType: string;

  @AllowNull(false)
  @Column
  origin: 'Data Source' | 'Data Stream';

  @AllowNull(true)
  @Column({ field: 'origin_id' })
  originId: string;

  @AllowNull(false)
  @Column
  destination: DataStreamDestination;

  @AllowNull(true)
  @Column({ field: 'cloud_type' })
  cloudType: CloudType;

  @AllowNull(true)
  @Column({ field: 'cloud_creds_id' })
  cloudCredsId: string;

  @AllowNull(true)
  @Column({ field: 'aws_cloud_region' })
  awsCloudRegion: AWS_REGION;

  @AllowNull(true)
  @Column({ field: 'gcp_cloud_region' })
  gcpCloudRegion: GCP_REGION;

  @AllowNull(true)
  @Column({ field: 'edge_stream_type' })
  edgeStreamType: EdgeStreamType;

  @AllowNull(true)
  @Column({ field: 'aws_stream_type' })
  awsStreamType: AWSStreamType;

  @AllowNull(true)
  @Column({ field: 'gcp_stream_type' })
  gcpStreamType: GCPStreamType;

  @AllowNull(false)
  @Column
  size: number;

  @AllowNull(false)
  @Column({
    type: DataType.BOOLEAN,
    field: 'enable_sampling',
  })
  enableSampling: boolean;

  @AllowNull(true)
  @Column({ field: 'sampling_interval' })
  samplingInterval: number;

  @AllowNull(false)
  @Column({ field: 'transformation_args_list', type: JSON_TYPE })
  transformationArgsList: TransformationArgs[];

  @AllowNull(false)
  @Column({ field: 'data_retention', type: JSON_TYPE })
  dataRetention: RetentionInfo[];

  @AllowNull(true)
  @Column({ field: 'project_id' })
  projectId: string;

  @AllowNull(true)
  @Column({ field: 'end_point' })
  endPoint: string;
}
// hack to work around sequelize bug
(<any>DataStreamModel.prototype)._options = {};
