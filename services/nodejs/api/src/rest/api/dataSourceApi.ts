import { DocType } from '../model/baseModel';
import { DataSource } from '../model/dataSource';
import { CategoryInfo } from '../model/category';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllDataSources(tenantId): Promise<DataSource[]> {
  return getDBService().getAllDocuments<DataSource>(
    tenantId,
    DocType.DataSource
  );
}

export async function getAllDataSourcesForEdge(
  tenantId,
  edgeId
): Promise<DataSource[]> {
  return getDBService().getAllDocumentsForEdge<DataSource>(
    tenantId,
    edgeId,
    DocType.DataSource
  );
}
