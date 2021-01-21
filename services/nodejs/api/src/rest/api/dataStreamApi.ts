import { DocType } from '../model/baseModel';
import { DataStream } from '../model/dataStream';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllDataStreams(tenantId): Promise<DataStream[]> {
  return getDBService().getAllDocuments<DataStream>(
    tenantId,
    DocType.DataStream
  );
}
