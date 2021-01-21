import {
  Table,
  Column,
  Model,
  PrimaryKey,
  CreatedAt,
  UpdatedAt,
  AllowNull,
  DataType,
} from 'sequelize-typescript';

@Table({ timestamps: true, tableName: 'edge_model' })
export class EdgeModel extends Model<EdgeModel> {
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
  @Column({ field: 'serial_number' })
  serialNumber: string;

  @AllowNull(false)
  @Column({ field: 'ip_address' })
  ipAddress: string;

  @AllowNull(false)
  @Column
  gateway: string;

  @AllowNull(false)
  @Column
  subnet: string;

  @AllowNull(false)
  @Column({ field: 'edge_devices' })
  edgeDevices: number;

  @AllowNull(false)
  @Column({ field: 'storage_capacity' })
  storageCapacity: number;

  @AllowNull(false)
  @Column({ field: 'storage_usage' })
  storageUsage: number;

  @AllowNull(true)
  @Column({
    type: DataType.BOOLEAN,
  })
  connected: boolean;
}
// hack to work around sequelize bug
(<any>EdgeModel.prototype)._options = {};
