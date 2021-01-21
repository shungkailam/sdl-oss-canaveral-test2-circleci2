import {
  Table,
  Column,
  Model,
  PrimaryKey,
  CreatedAt,
  UpdatedAt,
  AllowNull,
} from 'sequelize-typescript';

@Table({ timestamps: true, tableName: 'sensor_model' })
export class SensorModel extends Model<SensorModel> {
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
  @Column({ field: 'edge_id' })
  edgeId: string;

  @AllowNull(false)
  @Column({ field: 'topic_name' })
  topicName: string;
}
// hack to work around sequelize bug
(<any>SensorModel.prototype)._options = {};
