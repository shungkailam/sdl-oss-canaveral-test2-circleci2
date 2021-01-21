import {
  Table,
  Column,
  Model,
  AllowNull,
  DataType,
} from 'sequelize-typescript';

@Table({ timestamps: false, tableName: 'data_source_field_selector_model' })
export class DataSourceFieldSelectorModel extends Model<
  DataSourceFieldSelectorModel
> {
  @AllowNull(false)
  @Column({ field: 'data_source_id' })
  dataSourceId: string;

  @AllowNull(false)
  @Column({ field: 'field_id', type: DataType.INTEGER })
  fieldId: number;

  @AllowNull(false)
  @Column({ field: 'category_value_id', type: DataType.INTEGER })
  categoryValueId: number;
}
