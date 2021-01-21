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
import { ScriptParam } from '../model/baseModel';
import { getJsonType } from '../util/dbUtil';

const JSON_TYPE = getJsonType();

@Table({ timestamps: true, tableName: 'script_model' })
export class ScriptModel extends Model<ScriptModel> {
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
  @Column
  type: 'Transformation' | 'Function';

  @AllowNull(false)
  @Column
  language: string;

  @AllowNull(false)
  @Column
  environment: string;

  @AllowNull(false)
  @Column(DataType.STRING(8192))
  code: string;

  @AllowNull(false)
  @Column(JSON_TYPE)
  params: ScriptParam[];

  @AllowNull(true)
  @Column({ field: 'project_id' })
  projectId: string;
}
// hack to work around sequelize bug
(<any>ScriptModel.prototype)._options = {};
