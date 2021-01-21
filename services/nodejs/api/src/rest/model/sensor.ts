import { EdgeBaseModel, EdgeBaseModelKeys } from './baseModel';

/**
 * For .NEXT Nice we do not have a way to identify a sensor (e.g., via certificate).
 * The sensor discovery service will make wildcard (#) subscription to
 * mqtt server and report each distinct mqtt topic as a sensor.
 */
export interface Sensor extends EdgeBaseModel {
  /**
   * mqtt topic name that identifies the sensor.
   */
  topicName: string;
  // type: string;
  // status or last_seen
}
export const SensorKeys = ['topicName'].concat(EdgeBaseModelKeys);
