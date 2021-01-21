import { messageService } from '../services/msg.service';
import {
  DeleteRequest,
  makeObjectRequestBase,
  MESSAGE_KEY,
  ObjectDeleteMessageEmitter,
  ObjectMessageEmitter,
  ObjectRequestBase,
  ResponseBase,
} from './base';
import { BaseModel, EdgeBaseModel } from '../model/baseModel';
import { ReportSensorsRequest, ReportSensorsResponse } from './reportSensors';
import { Sensor } from '../model/sensor';
import { LogUploadPayload } from '../model/log';
import { logger } from '../util/logger';

export interface ObjectChangeEmitter<T extends BaseModel> {
  (obj: T): Promise<ResponseBase>;
}
export interface ObjectDeleteEmitter {
  (tenantId: string, id: string): Promise<ResponseBase>;
}
export interface SensorsEmitter {
  (sensors: Sensor[]): Promise<ResponseBase>;
}
export interface RequestLogUploadEmitter {
  (payload: LogUploadPayload): Promise<ResponseBase>;
}
export function getTenantObjectChangeEmitter<T extends BaseModel>(
  msg: MESSAGE_KEY
): ObjectChangeEmitter<T> {
  return (obj: T) => {
    logger.debug(`getTenantObjectChangeEmitter, msg=${msg}`);
    const emitter: ObjectMessageEmitter<T> = messageService.getEmitterToTenant<
      ObjectRequestBase<T>,
      ResponseBase
    >(obj.tenantId, msg);
    if (emitter) {
      const req: ObjectRequestBase<T> = makeObjectRequestBase<T>(
        obj.tenantId,
        obj
      );
      return emitter(req);
    } else {
      logger.info(
        `getTenantObjectChangeEmitter, no emitter for msg=${msg}`,
        obj
      );
    }
    return null;
  };
}
export function getTenantObjectDeleteEmitter(
  msg: MESSAGE_KEY
): ObjectDeleteEmitter {
  return (tenantId: string, id: string) => {
    logger.debug(`getTenantObjectDeleteEmitter, msg=${msg}`);
    const emitter: ObjectDeleteMessageEmitter = messageService.getEmitterToTenant<
      DeleteRequest,
      ResponseBase
    >(tenantId, msg);
    if (emitter) {
      const req: DeleteRequest = {
        tenantId,
        id,
      };
      return emitter(req);
    } else {
      logger.info(
        `getTenantObjectDeleteEmitter, no emitter for msg=${msg}, tenantId=${tenantId}, id=${id}`
      );
    }
    return null;
  };
}
export function getEdgeObjectChangeEmitter<T extends EdgeBaseModel>(
  msg: MESSAGE_KEY
): ObjectChangeEmitter<T> {
  return (obj: T) => {
    logger.debug(`getEdgeObjectChangeEmitter, msg=${msg}`);
    const emitter: ObjectMessageEmitter<T> = messageService.getEmitterToEdge<
      ObjectRequestBase<T>,
      ResponseBase
    >(obj.tenantId, obj.edgeId, msg);
    if (emitter) {
      const req: ObjectRequestBase<T> = makeObjectRequestBase<T>(
        obj.tenantId,
        obj
      );
      return emitter(req);
    } else {
      logger.info(`getEdgeObjectChangeEmitter, no emitter for msg=${msg}`, obj);
    }
    return null;
  };
}
export function getEdgeObjectDeleteEmitter(
  msg: MESSAGE_KEY,
  edgeId: string
): ObjectDeleteEmitter {
  return (tenantId: string, id: string) => {
    logger.debug(`getEdgeObjectDeleteEmitter, msg=${msg}`);
    const emitter: ObjectDeleteMessageEmitter = messageService.getEmitterToEdge<
      DeleteRequest,
      ResponseBase
    >(tenantId, edgeId, msg);
    if (emitter) {
      const req: DeleteRequest = {
        tenantId,
        id,
      };
      return emitter(req);
    } else {
      logger.info(
        `getEdgeObjectDeleteEmitter, no emitter for msg=${msg}, tenantId=${tenantId}, id=${id}`
      );
    }
    return null;
  };
}
export function getSensorChangeEmitter(): ObjectChangeEmitter<Sensor> {
  return (sensor: Sensor) => {
    logger.debug(`getSensorChangeEmitter`, sensor);
    const tenantId = sensor.tenantId;
    const emitter = messageService.getEmitterToServer<
      ReportSensorsRequest,
      ReportSensorsResponse
    >('reportSensors');
    if (emitter) {
      const req: ReportSensorsRequest = {
        tenantId,
        sensors: [sensor],
      };
      return emitter(req);
    } else {
      logger.info(`getSensorChangeEmitter - no emitter`, sensor);
    }
    return null;
  };
}
export function getLogUploadRequestEmitter(
  tenantId: string,
  edgeId: string
): RequestLogUploadEmitter {
  return (payload: LogUploadPayload) => {
    const emitter = messageService.getEmitterToEdge<
      LogUploadPayload,
      ResponseBase
    >(tenantId, edgeId, 'logUpload');
    if (emitter) {
      return emitter(payload);
    } else {
      logger.info(`getLogUploadRequestEmitter - no emitter`, payload);
    }
    return null;
  };
}
