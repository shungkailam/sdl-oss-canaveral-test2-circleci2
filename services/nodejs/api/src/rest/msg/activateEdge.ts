import {
  RequestBase,
  ResponseBase,
  MessageHandler,
  MessageEmitter,
} from './base';

//////////////////////////////////////////////////////
//
// Edge API
//
//////////////////////////////////////////////////////

export interface ActivateEdgeRequest extends RequestBase {
  edgeId: string;
  serialNumber: string;
}
export interface ActivateEdgeResponse extends ResponseBase {}
// MESSAGE('activateEdge')
export type ActivateEdge = MessageHandler<
  ActivateEdgeRequest,
  ActivateEdgeResponse
>;
export type ActivateEdgeEmitter = MessageEmitter<
  ActivateEdgeRequest,
  ActivateEdgeResponse
>;
