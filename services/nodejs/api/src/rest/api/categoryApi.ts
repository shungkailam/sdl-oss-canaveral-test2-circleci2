import { DocType } from '../model/baseModel';
import { Category } from '../model/category';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllCategories(tenantId: string): Promise<Category[]> {
  return getDBService().getAllDocuments<Category>(tenantId, DocType.Category);
}
