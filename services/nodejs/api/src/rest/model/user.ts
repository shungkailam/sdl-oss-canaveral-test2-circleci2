import { BaseModel, BaseModelKeys } from './baseModel';

export enum UserRoleType {
  INFRA_ADMIN = 'INFRA_ADMIN',
  USER = 'USER',
  OPERATOR = 'OPERATOR',
  OPERATOR_TENANT = 'OPERATOR_TENANT',
}

/**
 * User
 * User of Sherlock system.
 */
export interface User extends BaseModel {
  /**
   * Email of user
   */
  email: string;
  /**
   * User name
   */
  name: string;
  /**
   * SHA-256 hash of user password
   */
  password: string;

  role: UserRoleType;
}
export const UserKeys = ['email', 'name', 'password', 'role'].concat(
  BaseModelKeys
);
