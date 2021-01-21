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
import { CategoryInfo } from '../model/category';
import { getJsonType } from '../util/dbUtil';

const JSON_TYPE = getJsonType();

@Table({ timestamps: true, tableName: 'project_model' })
export class ProjectModel extends Model<ProjectModel> {
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

  @Column description: string;

  // @AllowNull(false)
  // @Column(JSON_TYPE)
  // cloudCredentialIds: string[];

  // @AllowNull(false)
  // @Column(JSON_TYPE)
  // userIds: string[];

  @AllowNull(false)
  @Column({ field: 'edge_selector_type' })
  edgeSelectorType: 'Category' | 'Explicit';

  @Column({ field: 'edge_selectors', type: JSON_TYPE })
  edgeSelectors?: CategoryInfo[];
}
// hack to work around sequelize bug
(<any>ProjectModel.prototype)._options = {};
