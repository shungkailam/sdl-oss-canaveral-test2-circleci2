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

@Table({ timestamps: true, tableName: 'category_model' })
export class CategoryModel extends Model<CategoryModel> {
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

  @Column purpose: string;
}
// hack to work around sequelize bug
(<any>CategoryModel.prototype)._options = {};
