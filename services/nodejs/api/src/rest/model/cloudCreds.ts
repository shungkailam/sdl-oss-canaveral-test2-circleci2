import { BaseModel, BaseModelKeys, CloudType } from './baseModel';

/**
 * GCP credential.
 * Service account info json.
 */
export interface GCPCredential {
  type: string;
  project_id: string;
  private_key_id: string;
  private_key: string;
  client_email: string;
  client_id: string;
  auth_uri: string;
  token_uri: string;
  auth_provider_x509_cert_url: string;
  client_x509_cert_url: string;
}
/**
 * AWS credential.
 */
export interface AWSCredential {
  accessKey: string;
  secret: string;
}
/**
 * Cloud Data Service Account Credentials.
 * Since Sherlock cloudmgmt is not yet secure,
 * we do not store this in cloudmgmt.
 * This is currently only exposed on the Edge side
 * to make progress on sending data from Edge to Cloud
 * for hackathon and .NEXT Nice.
 */
export interface CloudCreds extends BaseModel {
  /**
   * Cloud type for this cloud cred.
   */
  type: CloudType;
  /**
   * Name for the cloud cred.
   */
  name: string;
  /**
   * Description for the cloud cred.
   */
  description: string;
  // the following representation for credential is not ideal,
  // but we are constrained by what tsoa supports
  /**
   * Credential for the cloud creds.
   * Required when type == AWS.
   */
  awsCredential?: string;
  /**
   * Credential for the cloud creds.
   * Required when type == GCP.
   */
  gcpCredential?: string;

  /**
   * Indicate whether the data is encrypted
   */
  iflagEncrypted?: boolean;
}
export const CloudCredsKeys = [
  'type',
  'name',
  'description',
  'awsCredential',
  'gcpCredential',
].concat(BaseModelKeys);
