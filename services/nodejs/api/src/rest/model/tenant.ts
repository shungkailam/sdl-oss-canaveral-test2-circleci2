/**
 * A tenant represents a customer account.
 * A tenant may have multiple edges.
 * Every object in DB belonging to a tenant
 * will have a tenantId field.
 * Tenant object, like every other object
 * in DB will have id and version fields.
 * The id and version fields are marked as optional
 * because they are not required in create operation.
 */
export interface Tenant {
  /**
   * Unique id to identify the tenant.
   * This could be supplied during create or DB generated.
   * For Nice we will have fixed tenant id such as
   *   tenant-id-waldot
   *   tenant-id-rocket-blue
   */
  id?: string;
  /**
   * Version number of object maintained by DB.
   * Not currently used.
   */
  version?: number;
  /**
   * Name of the tenant.
   * E.g., WalDot or Rocket Blue, etc.
   */
  name: string;
  /**
   * Unique token for tenant.
   * Used in authentication.
   */
  token: string;

  description?: string;
}

export const TenantKeys = ['id', 'version', 'name', 'token'];
