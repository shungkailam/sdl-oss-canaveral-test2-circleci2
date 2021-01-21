import { BaseModel } from './baseModel';

export interface DockerProfile extends BaseModel {
  name: string;
  description: string;
  cloudCredsID?: string;
  type: 'AWS' | 'GCP' | 'Azure' | 'ContainerRegistry';
  server: string;
  userName: string;
  email: string;
  pwd: string;
  credentials: string;
  /**
   * Indicate whether the data is encrypted
   */
  iflagEncrypted?: boolean;
}
