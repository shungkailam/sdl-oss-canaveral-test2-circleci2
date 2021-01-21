import { EdgeBaseModel } from './baseModel';

export interface LogUploadPayload {
  url: string;
}

export interface LogUploadCompletePayload {
  url: string;
  status: LogUploadStatus;
  errorMessage?: string;
}

// Log upload request from the UI
export interface RequestLogUploadPayload {
  edgeIds: string[];
}

export interface RequestLogDownloadPayload {
  location: string;
}

export enum LogUploadStatus {
  PENDING = 'PENDING',
  SUCCESS = 'SUCCESS',
  FAILED = 'FAILED',
  TIMEDOUT = 'TIMEDOUT',
}

/**
 * A log entry describes the metadata for a log bundle
 * from an edge collected in part of a batch for a given tenant.
 */
export interface LogEntry extends EdgeBaseModel {
  /**
   * id that identifies logs from different edge as the same batch.
   */
  batchId: string;
  /**
   * Location or object key for the log in the bucket.
   */
  location: string;
  /**
   * Creation timestamp
   */
  createdAt: Date;
  /**
   * Status of this log entry.
   */
  status: LogUploadStatus;
  /**
   * Error message - optional, should be populated when status == 'FAILED'
   */
  errorMessage?: string;
}
