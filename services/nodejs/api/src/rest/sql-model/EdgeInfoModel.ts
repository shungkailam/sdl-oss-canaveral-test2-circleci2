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

@Table({ timestamps: true, tableName: 'edge_info_model' })
export class EdgeInfoModel extends Model<EdgeInfoModel> {
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

  @Column({ field: 'num_cpu' })
  NumCPU: string;

  @Column({ field: 'total_memory_kb' })
  TotalMemoryKB: string;

  @Column({ field: 'total_storage_kb' })
  TotalStorageKB: string;

  @Column({ field: 'gpu_info' })
  GPUInfo: string;

  @Column({ field: 'cpu_usage' })
  CPUUsage: string;

  @Column({ field: 'memory_free_kb' })
  MemoryFreeKB: string;

  @Column({ field: 'storage_free_kb' })
  StorageFreeKB: string;

  @Column({ field: 'gpu_usage' })
  GPUUsage: string;
}
// hack to work around sequelize bug
(<any>EdgeInfoModel.prototype)._options = {};
