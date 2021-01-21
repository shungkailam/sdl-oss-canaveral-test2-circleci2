import { DocType } from '../model/baseModel';
import { CloudCreds } from '../model/cloudCreds';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllCloudCreds(
  tenantId: string
): Promise<CloudCreds[]> {
  return getDBService().getAllDocuments<CloudCreds>(
    tenantId,
    DocType.CloudCreds
  );
}
