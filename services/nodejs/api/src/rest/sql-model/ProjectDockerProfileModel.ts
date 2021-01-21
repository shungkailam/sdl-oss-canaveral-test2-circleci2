import { Table, Column, Model, AllowNull } from 'sequelize-typescript';

@Table({ timestamps: false, tableName: 'project_docker_profile_model' })
export class ProjectDockerProfileModel extends Model<
  ProjectDockerProfileModel
> {
  @AllowNull(false)
  @Column({ field: 'project_id' })
  projectId: string;

  @AllowNull(false)
  @Column({ field: 'docker_profile_id' })
  dockerProfileId: string;
}
