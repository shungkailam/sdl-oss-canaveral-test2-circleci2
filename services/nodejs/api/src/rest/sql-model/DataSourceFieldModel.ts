import { Table, Column, Model, AllowNull } from 'sequelize-typescript';

@Table({ timestamps: false, tableName: 'data_source_field_model' })
export class DataSourceFieldModel extends Model<DataSourceFieldModel> {
  @AllowNull(false)
  @Column({ field: 'data_source_id' })
  dataSourceId: string;

  @AllowNull(false)
  @Column
  name: string;

  @AllowNull(true)
  @Column({ field: 'mqtt_topic' })
  mqttTopic: string;

  @AllowNull(true)
  @Column({ field: 'field_type' })
  fieldType: string;
}
