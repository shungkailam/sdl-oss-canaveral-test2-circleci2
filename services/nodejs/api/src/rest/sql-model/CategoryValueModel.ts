import { Table, Column, Model, AllowNull } from 'sequelize-typescript';

@Table({ timestamps: false, tableName: 'category_value_model' })
export class CategoryValueModel extends Model<CategoryValueModel> {
  @AllowNull(false)
  @Column({ field: 'category_id' })
  categoryId: string;

  @AllowNull(false)
  @Column
  value: string;
}
