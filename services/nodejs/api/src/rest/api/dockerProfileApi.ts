import { DocType } from '../model/baseModel';
import { DockerProfile } from '../model/dockerProfile';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllDockerProfiles(
  tenantId: string
): Promise<DockerProfile[]> {
  return getDBService().getAllDocuments<DockerProfile>(
    tenantId,
    DocType.DockerProfile
  );
}
