/////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////
//                                                         //
//  Sherlock Cloud <-> Edge management Messaging protocol  //
//                                                         //
/////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////

// Out messaging protocol follows a request / response format
// based on the following pattern supported by Socket.IO:

import { logger } from '../util/logger';

/*
//
// server
//
var io = require('socket.io')();

io.on('connection', function (socket) {
  socket.on('ferret', function (name, fn) {
    fn('woot');
  });
});

//
// client
//
var socket = io();
socket.on('connect', function () {
  socket.emit('ferret', 'tobi', function (data) {
    console.log(data); // data will be 'woot'
  });
});

 */

export type MESSAGE_KEY =
  | 'reportEdge'
  | 'reportSensors'
  | 'activateEdge'
  | 'deactivateEdge'
  | 'onCreateCategory'
  | 'onDeleteCategory'
  | 'onUpdateCategory'
  | 'onCreateDataSource'
  | 'onDeleteDataSource'
  | 'onUpdateDataSource'
  | 'onCreateScript'
  | 'onDeleteScript'
  | 'onUpdateScript'
  | 'onCreateDataStream'
  | 'onDeleteDataStream'
  | 'onUpdateDataStream'
  | 'onCreateCloudCreds'
  | 'onDeleteCloudCreds'
  | 'onUpdateCloudCreds'
  | 'logUpload'
  | 'logUploadComplete';

export enum NotificationTopics {
  reportEdge = 'reportEdge',
  reportSensors = 'reportSensors',
  onCreateCategory = 'onCreateCategory',
  onDeleteCategory = 'onDeleteCategory',
  onUpdateCategory = 'onUpdateCategory',
  onCreateDataSource = 'onCreateDataSource',
  onDeleteDataSource = 'onDeleteDataSource',
  onUpdateDataSource = 'onUpdateDataSource',
  onCreateScript = 'onCreateScript',
  onDeleteScript = 'onDeleteScript',
  onUpdateScript = 'onUpdateScript',
  onCreateDataStream = 'onCreateDataStream',
  onDeleteDataStream = 'onDeleteDataStream',
  onUpdateDataStream = 'onUpdateDataStream',
  onCreateCloudCreds = 'onCreateCloudCreds',
  onDeleteCloudCreds = 'onDeleteCloudCreds',
  onUpdateCloudCreds = 'onUpdateCloudCreds',
  logUpload = 'logUpload',
  logUploadComplete = 'logUploadComplete',
}

type Socket = any;

export interface RequestBase {
  tenantId: string;
}

export interface DeleteRequest extends RequestBase {
  id: string;
}

export interface ObjectRequestBase<T> extends RequestBase {
  doc: T;
}

export interface ResponseBase {
  statusCode: number;
  message?: string;
}

export interface GenericCallback<R> {
  (response: R): void;
}

export interface MessageHandler<Req, Resp> {
  (socket: Socket, request: Req): Promise<Resp>;
}

export interface MessageEmitter<Req, Resp> {
  (request: Req): Promise<Resp>;
}

export type ObjectMessageHandler<T> = MessageHandler<
  ObjectRequestBase<T>,
  ResponseBase
>;
export type ObjectMessageEmitter<T> = MessageEmitter<
  ObjectRequestBase<T>,
  ResponseBase
>;
export function makeObjectRequestBase<T>(
  tenantId: string,
  doc: T
): ObjectRequestBase<T> {
  return {
    tenantId,
    doc,
  };
}
export type ObjectDeleteMessageHandler = MessageHandler<
  DeleteRequest,
  ResponseBase
>;
export type ObjectDeleteMessageEmitter = MessageEmitter<
  DeleteRequest,
  ResponseBase
>;

// wrapper for socket.on that consumes Promise
// Idiom for server to add message handler:
//   on(socket, '<msg_key>', handler)
// where handler is simply
//   (socket, req): Promise<resp>
export function on<Q, R>(
  socket: any,
  msg: MESSAGE_KEY,
  handler: MessageHandler<Q, R>
) {
  socket.on(msg, function(req: Q, cb: GenericCallback<R>) {
    logger.info({ req }, `socket.on ${msg}`);
    handler(socket, req).then(
      resp => {
        logger.info({ req, resp }, `socket.on ${msg} success`);
        if (cb) {
          cb(resp);
        }
      },
      err => {
        logger.info({ req, err }, `socket.on ${msg} error`);
        const r: any = {
          message: `${msg} handler failed [${err}]`,
          statusCode: 500,
        };
        if (cb) {
          cb(r);
        }
      }
    );
  });
}

const EMIT_TIMEOUT_MILLIS = 30000;

// wrapper for socket.emit that returns Promise
// Idiom for client to emit a message:
//   emit(socket, '<msg_key>')(req).then(resp => ...)
export function emit<Q, R>(
  socket: any,
  msg: MESSAGE_KEY
): MessageEmitter<Q, R> {
  return (req: Q) => {
    return new Promise<R>((resolve, reject) => {
      const timer = setTimeout(() => {
        reject(Error('Timeout waiting for emit response'));
      }, EMIT_TIMEOUT_MILLIS);
      logger.info({ req }, `socket.emit ${msg}`);
      socket.emit(msg, req, function(resp: R) {
        logger.info({ req, resp }, `socket.emit ${msg} done`);
        clearTimeout(timer);
        resolve(resp);
      });
    });
  };
}

/**
 * Emit message to all edges of a tenant.
 * @param io Server socket.io instance
 * @param {string} tenantId
 * @param {MESSAGE_KEY} msg
 * @returns {MessageEmitter<Q, R>}
 */
export function emitToTenant<Q, R>(
  io: any,
  tenantId: string,
  msg: MESSAGE_KEY
): MessageEmitter<Q, R> {
  return (req: Q) => {
    return new Promise<R>((resolve, reject) => {
      // callback not supported in broadcast for socket.io
      logger.info({ req, tenantId }, `socket.broadcast ${msg}`);
      io.to(tenantId).emit(msg, req);
      resolve(null);
    });
  };
}
