import { ReportLogUploadComplete } from '../rest/msg/index';

import { logger } from '../rest/util/logger';
import { completeUpload } from '../rest/api/logApi';
import { LogUploadCompletePayload } from '../rest/model';

// After the log upload to the signed S3 URL is completed by the edge,
// it reports reports the log completion with the upload status.
export const reportLogUploadComplete: ReportLogUploadComplete = async (
  socket: SocketIO.Socket,
  payload: LogUploadCompletePayload
) => {
  try {
    logger.info(payload, '>>> reportLogUploadComplete reported: ');
    let tenantId = null;
    try {
      const tokens = payload.url.split(/[/?]/);
      tenantId = tokens[4];
    } catch (err) {
      throw Error('reportLogUploadComplete: Bad input!');
    }
    await completeUpload(
      tenantId,
      payload.url,
      payload.status,
      payload.errorMessage
    );
    return {
      statusCode: 200,
    };
  } catch (e) {
    logger.error(e, '>>> reportLogUploadComplete: caught exception:');
    return {
      doc: null,
      statusCode: 500,
    };
  }
};
