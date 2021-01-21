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

@Table({ timestamps: true, tableName: 'docker_profile_model' })
export class DockerProfileModel extends Model<DockerProfileModel> {
  @PrimaryKey
  @Column
  id: string;

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
  description: string;

  @AllowNull(true)
  @Column({ field: 'cloud_creds_id' })
  cloudCredsID: string;

  @AllowNull(false)
  @Column
  type: string;

  @AllowNull(false)
  @Column
  server: string;

  @AllowNull(false)
  @Column({ field: 'user_name' })
  userName: string;

  @AllowNull(false)
  @Column
  email: string;

  @AllowNull(false)
  @Column
  pwd: string;

  @Column credentials: string;

  @CreatedAt created_at: Date;

  @UpdatedAt updated_at: Date;

  @AllowNull(true)
  @Column({ field: 'iflag_encrypted', type: DataType.BOOLEAN })
  iflagEncrypted: boolean;
}
// hack to work around sequelize bug
(<any>DockerProfileModel.prototype)._options = {};
