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
import { getJsonType } from '../util/dbUtil';

const JSON_TYPE = getJsonType();

@Table({
  timestamps: true,
  tableName: 'log_model',
})
export class LogModel extends Model<LogModel> {
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
  @Column({ field: 'batch_id' })
  batchId: string;

  // This is unique, no overwriting which makes it
  // easy to identify the upload on completion
  @AllowNull(false)
  @Column
  location: string;

  @AllowNull(false)
  @Column
  status: string;

  @Column({ field: 'error_message' })
  errorMessage: string;
}
// hack to work around sequelize bug
(<any>LogModel.prototype)._options = {};
