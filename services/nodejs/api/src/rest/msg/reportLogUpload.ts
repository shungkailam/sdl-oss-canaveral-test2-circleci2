import { ResponseBase, MessageHandler } from './base';
import { LogUploadCompletePayload } from '../model/log';

//////////////////////////////////////////////////////
//
// Cloud API
//
//////////////////////////////////////////////////////

// To the cloud
export type ReportLogUploadComplete = MessageHandler<
  LogUploadCompletePayload,
  ResponseBase
>;
