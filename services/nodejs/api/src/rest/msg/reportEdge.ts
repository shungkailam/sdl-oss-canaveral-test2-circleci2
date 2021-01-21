import { ResponseBase, MessageHandler, ObjectRequestBase } from './base';
import { Edge } from '../model/edge';

//////////////////////////////////////////////////////
//
// Cloud API
//
//////////////////////////////////////////////////////

export interface ReportEdgeRequest extends ObjectRequestBase<Edge> {}

export interface ReportEdgeResponse extends ResponseBase {
  doc: Edge;
}

// MESSAGE('reportEdge')
export type ReportEdge = MessageHandler<ReportEdgeRequest, ReportEdgeResponse>;
