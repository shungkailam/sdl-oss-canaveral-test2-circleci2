import {
  Table,
  Column,
  Model,
  PrimaryKey,
  CreatedAt,
  UpdatedAt,
  AllowNull,
} from 'sequelize-typescript';

@Table({ timestamps: true, tableName: 'script_runtime_model' })
export class ScriptRuntimeModel extends Model<ScriptRuntimeModel> {
  @PrimaryKey
  @Column
  id: string;

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
  language: string;

  @AllowNull(false)
  @Column
  builtin: boolean;

  @AllowNull(true)
  @Column({ field: 'docker_repo_uri' })
  dockerRepoURI: string;

  @AllowNull(true)
  @Column({ field: 'docker_profile_id' })
  dockerProfileID: string;

  @AllowNull(true)
  @Column
  dockerfile: string;

  @AllowNull(true)
  @Column({ field: 'project_id' })
  projectId: string;

  @CreatedAt created_at: Date;

  @UpdatedAt updated_at: Date;
}
// hack to work around sequelize bug
(<any>ScriptRuntimeModel.prototype)._options = {};
