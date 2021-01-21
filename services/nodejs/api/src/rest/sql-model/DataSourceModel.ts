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

import {
  DataSourceFieldInfo,
  DataSourceFieldSelector,
} from '../model/dataSource';
import { getJsonType } from '../util/dbUtil';

const JSON_TYPE = getJsonType();

@Table({ timestamps: true, tableName: 'data_source_model' })
export class DataSourceModel extends Model<DataSourceModel> {
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
  @Column({ field: 'edge_id' })
  edgeId: string;

  @AllowNull(false)
  @Column
  name: string;

  @AllowNull(false)
  @Column
  type: 'Sensor' | 'Gateway';

  @AllowNull(false)
  @Column({ field: 'sensor_model' })
  sensorModel: string;

  @AllowNull(false)
  @Column
  connection: 'Secure' | 'Unsecure';

  @AllowNull(false)
  @Column
  protocol: 'MQTT' | 'RTSP' | 'GIGEVISION' | 'OTHER';

  @AllowNull(false)
  @Column({ field: 'auth_type' })
  authType: 'CERTIFICATE' | 'PASSWORD' | 'TOKEN';
}
// hack to work around sequelize bug
(<any>DataSourceModel.prototype)._options = {};
