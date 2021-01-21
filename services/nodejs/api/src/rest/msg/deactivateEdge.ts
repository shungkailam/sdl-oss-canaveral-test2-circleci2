import { RequestBase, ResponseBase, MessageHandler } from './base';

//////////////////////////////////////////////////////
//
// Edge API
//
//////////////////////////////////////////////////////

export interface DeactivateEdgeRequest extends RequestBase {
  edgeId: string;
  edgeSerialNumber: string;
}
export interface DeactivateEdgeResponse extends ResponseBase {}
// MESSAGE('deactivateEdge')
export type DeactivateEdge = MessageHandler<
  DeactivateEdgeRequest,
  DeactivateEdgeResponse
>;
