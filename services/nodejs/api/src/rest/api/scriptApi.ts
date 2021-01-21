import { DocType } from '../model/baseModel';
import { Script } from '../model/index';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllScripts(tenantId): Promise<Script[]> {
  return getDBService().getAllDocuments<Script>(tenantId, DocType.Script);
}
