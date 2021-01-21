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
import { CloudType, AWSCredential, GCPCredential } from '../model/index';
import { getJsonType } from '../util/dbUtil';

const JSON_TYPE = getJsonType();

@Table({ timestamps: true, tableName: 'cloud_creds_model' })
export class CloudCredsModel extends Model<CloudCredsModel> {
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

  @AllowNull(false)
  @Column
  type: CloudType;

  @Column description: string;

  @AllowNull(true)
  @Column({ field: 'aws_credential' })
  awsCredential: string;

  @AllowNull(true)
  @Column({ field: 'gcp_credential' })
  gcpCredential: string;

  @AllowNull(true)
  @Column({ field: 'iflag_encrypted', type: DataType.BOOLEAN })
  iflagEncrypted: boolean;
}
// hack to work around sequelize bug
(<any>CloudCredsModel.prototype)._options = {};
