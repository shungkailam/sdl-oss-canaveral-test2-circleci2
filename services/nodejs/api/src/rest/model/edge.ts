import { BaseModel, BaseModelKeys } from './baseModel';

/**
 * Payload used in getBySerialNumber API.
 */
export interface SerialNumberPayload {
  /**
   * serial number for the edge
   */
  serialNumber: string;
}
/**
 * Playload used in getEdgeHandle API.
 */
export interface GetHandlePayload {
  /**
   * unique secret token per edge.
   */
  token: string;
  /**
   * ID for the tenant
   */
  tenantId: string;
}
/**
 * An Edge is a Nutanix (Kubernetes) cluster for a tenant.
 */
export interface Edge extends BaseModel {
  /**
   * name for the edge
   */
  name: string;
  /**
   * name for the edge
   */
  description?: string;
  /**
   * serial number for the edge
   */
  serialNumber: string;
  /**
   * IP Address for the edge
   */
  ipAddress: string;
  /**
   * Gateway IP for the edge
   */
  gateway: string;
  /**
   * Subnet mask for the edge
   */
  subnet: string;
  /**
   * number of devices (nodes) in this edge
   */
  edgeDevices: number;
  /**
   * storage capacity in GB
   */
  storageCapacity: number;
  /**
   * storage usage in GB
   */
  storageUsage: number;
  /**
   * Whether the edge is currently connected to cloudmgmt.
   */
  connected?: boolean;
}
export const EdgeKeys = [
  'name',
  'serialNumber',
  'ipAddress',
  'gateway',
  'subnet',
  'edgeDevices',
  'storageCapacity',
  'storageUsage',
  'connected',
].concat(BaseModelKeys);
