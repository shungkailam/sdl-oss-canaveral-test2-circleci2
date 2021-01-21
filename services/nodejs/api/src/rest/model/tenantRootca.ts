export interface TenantRootCA {
  id?: string;

  version?: number;

  tenantId: string;

  certificate: string;

  privateKey: string;

  awsDataKey: string;
}
