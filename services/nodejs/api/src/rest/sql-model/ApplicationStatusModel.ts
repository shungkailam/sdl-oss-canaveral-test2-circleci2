import {
  Table,
  Column,
  Model,
  PrimaryKey,
  CreatedAt,
  UpdatedAt,
  AllowNull,
} from 'sequelize-typescript';
import { getJsonType } from '../util/dbUtil';

const JSON_TYPE = getJsonType();

@Table({ timestamps: true, tableName: 'application_status_model' })
export class ApplicationStatusModel extends Model<ApplicationStatusModel> {
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
  @Column({ field: 'application_id' })
  applicationId: string;

  @CreatedAt created_at: Date;

  @UpdatedAt updated_at: Date;

  @AllowNull(false)
  @Column({ field: 'app_status', type: JSON_TYPE })
  appStatus: any;
}
// hack to work around sequelize bug
(<any>ApplicationStatusModel.prototype)._options = {};
