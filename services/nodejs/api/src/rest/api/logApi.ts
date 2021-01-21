import { DeleteDocumentResponse, DocType } from '../model/baseModel';
import { LogEntry } from '../model/log';
import { LogUploadStatus } from '../model/log';
import { logger } from '../util/logger';
import { getDBService } from '../db-configurator/dbConfigurator';
import logService from '../services/log.service';

//TODO - change the bucket
const LOG_BUCKET_NAME =
  process.env.S3_LOG_BUCKET || 'sherlock-support-bundle-us-west-2';
const LOG_VERSION = process.env.LOG_VERSION || 'v1';
const LOG_FILE_EXT = process.env.LOG_FILE_EXT || '.tgz';

export async function getAllLogs(tenantId: string): Promise<LogEntry[]> {
  return getDBService().getAllDocuments<LogEntry>(tenantId, DocType.Log);
}

// If the callback is set, it is called with the { url }
export async function getUploadUrl(
  tenantId: string,
  batchId: string,
  edgeId: string,
  callback?
): Promise<string> {
  return new Promise<string>(async (resolve, reject) => {
    try {
      const current = new Date();
      const key = createS3Key(LOG_VERSION, tenantId, batchId, edgeId);
      const url = await logService.getUploadUrl(LOG_BUCKET_NAME, key);
      const doc = {
        tenantId,
        batchId,
        edgeId,
        location: key,
        status: LogUploadStatus.PENDING,
      };
      const response = await getDBService().createDocument(
        tenantId,
        DocType.Log,
        doc
      );
      logger.info(response, 'Created log uplaod URL');
      if (!!callback) {
        try {
          const response = await callback({ url });
          if (!response) {
            // Fail even when the callback is not found
            throw new Error('Callback response is null');
          }
        } catch (err) {
          logger.error(err, `Failed to to invoke callback for edge: ${edgeId}`);
          // Update the DB status
          await getDBService()
            .getDocument<LogEntry>(tenantId, response._id, DocType.Log)
            .then(async doc => {
              doc.status = LogUploadStatus.FAILED;
              doc.errorMessage = err.message;
              await getDBService().updateDocument(
                tenantId,
                doc.id,
                DocType.Log,
                doc
              );
            });
          // Propagate after handling the error
          throw err;
        }
      }
      resolve(url);
    } catch (err) {
      reject(err);
    }
  });
}

export async function completeUpload(
  tenantId: string,
  url: string,
  status: LogUploadStatus,
  errorMessage: string
): Promise<string> {
  return new Promise<string>(async (resolve, reject) => {
    try {
      const key = extractS3Key(url);
      const logEntry = await getDBService().findOneDocument<LogEntry>(
        tenantId,
        { location: key },
        DocType.Log
      );
      if (logEntry) {
        logEntry.status = status;
        logEntry.errorMessage = errorMessage;
        await getDBService().updateDocument(
          tenantId,
          logEntry.id,
          DocType.Log,
          logEntry
        );
      } else {
        throw new Error('No log found for location ' + key);
      }
      resolve(key);
    } catch (err) {
      reject(err);
    }
  });
}

export async function getDownloadUrl(
  tenantId: string,
  location: string
): Promise<string> {
  return new Promise<string>(async (resolve, reject) => {
    try {
      const key = await logService.getDownloadUrl(LOG_BUCKET_NAME, location);
      resolve(key);
    } catch (err) {
      reject(err);
    }
  });
}

export async function deleteLogByLocation(
  tenantId: string,
  location: string
): Promise<DeleteDocumentResponse> {
  return new Promise<DeleteDocumentResponse>(async (resolve, reject) => {
    try {
      const logEntry = await getDBService().findOneDocument<LogEntry>(
        tenantId,
        location,
        DocType.Log
      );
      if (logEntry) {
        // Delete from the DB first, so that it does not
        // show up on UI on partial failure.
        const doc = await getDBService().deleteDocument(
          tenantId,
          logEntry.id,
          DocType.Log
        );
        await logService.deleteObject(LOG_BUCKET_NAME, logEntry.location);
        resolve(doc);
      } else {
        throw new Error('No document found for location ' + location);
      }
    } catch (err) {
      reject(err);
    }
  });
}

export async function deleteLogById(
  tenantId: string,
  id: string
): Promise<DeleteDocumentResponse> {
  return new Promise<DeleteDocumentResponse>(async (resolve, reject) => {
    try {
      const logEntry = await getDBService().getDocument<LogEntry>(
        tenantId,
        id,
        DocType.Log
      );
      if (logEntry) {
        // Delete from the DB first, so that it does not
        // show up on UI on partial failure.
        const doc = await getDBService().deleteDocument(
          tenantId,
          id,
          DocType.Log
        );
        await logService.deleteObject(LOG_BUCKET_NAME, logEntry.location);
        resolve(doc);
      } else {
        throw new Error('No document found with id ' + id);
      }
    } catch (err) {
      reject(err);
    }
  });
}

// Returns key like tenantId/YYYY/MM/DD/batchId/edgeId/edgeId-batchId.zip
function createS3Key(
  version: string,
  tenantId: string,
  batchId: string,
  edgeId: string
): string {
  const date = new Date();
  const timestamp = date
    .toISOString()
    .substring(0, 10)
    .replace(/-/g, '/');
  const filename = [edgeId, batchId].join('-') + LOG_FILE_EXT;
  return [version, tenantId, timestamp, batchId, edgeId, filename].join('/');
}

// The URL looks like
// https://bucket.s3.us-west-2.amazonaws.com/v1/tenantId/2018/04/18/batchId/edge/edge-batchId.zip?AWSAccessKeyId=...
function extractS3Key(url: string): string {
  return url
    .split(/[/?]/)
    .slice(3, 11)
    .join('/');
}
