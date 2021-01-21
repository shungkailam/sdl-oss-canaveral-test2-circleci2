import { BaseModel } from './baseModel';
import { CategoryInfo } from './category';

export interface Application extends BaseModel {
  name: string;
  description?: string;
  yamlData: any;
  // originSelectors: CategoryInfo[];
  projectId: string;
  edgeIds?: string[];
}
