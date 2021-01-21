import { BaseModel, BaseModelKeys } from './baseModel';

/**
 * Category
 * Similar to labels for Kubernetes.
 * For Nice we only support limited (implicit) category composition
 * when a class is specified as a list of categories:
 *   OR - among different category values from the same key
 *   AND - among different category keys
 * Post Nice we should support more general (explicit) category compositions such as
 *   AND, OR, NOT
 * among sensible category combinations.
 */
export interface Category extends BaseModel {
  /**
   * Unique name that identifies a category.
   * E.g., Airport, Terminal, Floor, Environment, Department, etc.
   */
  name: string;
  /**
   * Purpose of the category.
   */
  purpose: string;
  /**
   * All allowed values for the category.
   * E.g.,
   *   SFO, ORD, LAX, ...
   *   1, 2, 3, ...
   *   Production, Dev, Test, ...
   *   Sales, HR, Eng, ...
   */
  values: string[];
}
export const CategoryKeys = ['name', 'purpose', 'values'].concat(BaseModelKeys);

/**
 * A CategoryInfo is a choice of a value from a category.
 */
export interface CategoryInfo {
  /**
   * The id of the category.
   * E.g., id for the Airport category
   */
  id: string;
  /**
   * The value chosen among allowed values from the category.
   * E.g., SFO
   */
  value: string;
}
export const CategoryInfoKeys = ['id', 'value'];
