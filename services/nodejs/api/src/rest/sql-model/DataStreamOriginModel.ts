import {
  Table,
  Column,
  Model,
  AllowNull,
  DataType,
} from 'sequelize-typescript';

@Table({ timestamps: false, tableName: 'data_stream_origin_model' })
export class DataStreamOriginModel extends Model<DataStreamOriginModel> {
  @AllowNull(false)
  @Column({ field: 'data_stream_id' })
  dataStreamId: string;

  @AllowNull(false)
  @Column({ field: 'category_value_id', type: DataType.INTEGER })
  categoryValueId: number;
}
