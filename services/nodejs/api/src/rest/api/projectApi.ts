import { DocType } from '../model/baseModel';
import { Project } from '../model/project';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllProjects(tenantId: string): Promise<Project[]> {
  return getDBService().getAllDocuments<Project>(tenantId, DocType.Project);
}
