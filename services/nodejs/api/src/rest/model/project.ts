import { BaseModel, BaseModelKeys } from './baseModel';
import { CategoryInfo } from './category';

export interface ProjectUserInfo {
  userId: string;
  role: string;
}
/**
 * A Project is logical grouping of resouces.
 * (Edges, CloudCreds, Users, DataStreams, etc.)
 */
export interface Project extends BaseModel {
  /**
   * name for the project
   */
  name: string;
  /**
   * description for the project
   */
  description: string;

  /**
   * List of ids of cloud credentials this project has access to.
   */
  cloudCredentialIds: string[];

  dockerProfileIds: string[];

  /**
   * List of ProjectUserInfo of users who have access to this project.
   */
  users: ProjectUserInfo[];

  /**
   * Type of edge selector. Either 'Category' or 'Explicit'
   * Specify whether edges belonging to this project are
   * given by edgeIds ('Explicit') or edgeSelectors ('Category').
   */
  edgeSelectorType: 'Category' | 'Explicit';

  /**
   * List of ids of edges belong to this project.
   * Only relevant when edgeSelectorType === 'Explicit'
   */
  edgeIds?: string[];

  /**
   * Edge selectors - CategoryInfo list.
   * Only relevant when edgeSelectorType === 'Category'
   */
  edgeSelectors?: CategoryInfo[];
}
export const ProjectKeys = [
  'name',
  'description',
  'cloudCredentialIds',
  'userIds',
  'edgeSelectorType',
  'edgeIds',
  'edgeSelectors',
].concat(BaseModelKeys);
