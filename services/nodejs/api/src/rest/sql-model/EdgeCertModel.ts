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

@Table({ timestamps: true, tableName: 'edge_cert_model' })
export class EdgeCertModel extends Model<EdgeCertModel> {
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
  @Column(DataType.STRING(8192))
  certificate: string;

  @AllowNull(false)
  @Column({ field: 'private_key', type: DataType.STRING(8192) })
  privateKey: string;

  @AllowNull(false)
  @Column({ field: 'client_certificate', type: DataType.STRING(8192) })
  clientCertificate: string;

  @AllowNull(false)
  @Column({ field: 'client_private_key', type: DataType.STRING(8192) })
  clientPrivateKey: string;

  @AllowNull(false)
  @Column({ field: 'edge_certificate', type: DataType.STRING(8192) })
  edgeCertificate: string;

  @AllowNull(false)
  @Column({ field: 'edge_private_key', type: DataType.STRING(8192) })
  edgePrivateKey: string;

  @AllowNull(false)
  @Column({
    type: DataType.BOOLEAN,
  })
  locked: boolean;
}
// hack to work around sequelize bug
(<any>EdgeCertModel.prototype)._options = {};
