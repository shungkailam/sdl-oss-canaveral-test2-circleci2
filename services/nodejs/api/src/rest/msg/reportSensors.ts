import {
  RequestBase,
  ResponseBase,
  MessageHandler,
  MessageEmitter,
} from './base';
import { Sensor } from '../model/sensor';

//////////////////////////////////////////////////////
//
// Cloud API
//
//////////////////////////////////////////////////////

export interface ReportSensorsRequest extends RequestBase {
  sensors: Sensor[];
}

export type ReportSensorsResponse = ResponseBase;

// Edge should subscribe to all sensors Kafka topic
// and push it to Cloud
export type ReportSensors = MessageHandler<
  ReportSensorsRequest,
  ReportSensorsResponse
>;
