import { Table, Column, Model, AllowNull } from 'sequelize-typescript';

@Table({ timestamps: false, tableName: 'project_user_model' })
export class ProjectUserModel extends Model<ProjectUserModel> {
  @AllowNull(false)
  @Column({ field: 'project_id' })
  projectId: string;

  @AllowNull(false)
  @Column({ field: 'user_id' })
  userId: string;

  @AllowNull(false)
  @Column({ field: 'user_role' })
  role: string;
}
