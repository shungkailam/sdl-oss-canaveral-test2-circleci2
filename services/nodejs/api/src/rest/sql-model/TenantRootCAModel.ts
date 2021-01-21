import {
  Table,
  Column,
  Model,
  PrimaryKey,
  CreatedAt,
  UpdatedAt,
  AllowNull,
} from 'sequelize-typescript';

@Table({
  timestamps: true,
  tableName: 'tenant_rootca_model',
})
export class TenantRootCAModel extends Model<TenantRootCAModel> {
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
  certificate: string;

  @AllowNull(false)
  @Column({ field: 'private_key' })
  privateKey: string;

  @AllowNull(false)
  @Column({ field: 'aws_data_key' })
  awsDataKey: string;
}
// hack to work around sequelize bug
(<any>TenantRootCAModel.prototype)._options = {};
