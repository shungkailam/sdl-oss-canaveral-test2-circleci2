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

@Table({
  timestamps: true,
  tableName: 'tenant_model',
})
export class TenantModel extends Model<TenantModel> {
  @PrimaryKey
  @Column
  id: string;

  @CreatedAt created_at: Date;

  @UpdatedAt updated_at: Date;

  @AllowNull(false)
  @Column
  version: number;

  @AllowNull(false)
  @Column
  name: string;

  @AllowNull(false)
  @Column
  token: string;

  @Column description: string;

  @AllowNull(true)
  @Column({ field: 'profile', type: JSON_TYPE })
  profile: any;
}
// hack to work around sequelize bug
(<any>TenantModel.prototype)._options = {};
