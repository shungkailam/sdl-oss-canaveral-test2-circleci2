export interface ApplicationStatus {
  version?: number;
  tenantId: string;
  edgeId: string;
  applicationId: string;
  appStatus: any;
  createdAt: string;
  updatedAt: string;
}
