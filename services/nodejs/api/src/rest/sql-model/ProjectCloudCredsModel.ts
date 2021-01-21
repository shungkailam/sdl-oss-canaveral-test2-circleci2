import { Table, Column, Model, AllowNull } from 'sequelize-typescript';

@Table({ timestamps: false, tableName: 'project_cloud_creds_model' })
export class ProjectCloudCredsModel extends Model<ProjectCloudCredsModel> {
  @AllowNull(false)
  @Column({ field: 'project_id' })
  projectId: string;

  @AllowNull(false)
  @Column({ field: 'cloud_creds_id' })
  cloudCredsId: string;
}
