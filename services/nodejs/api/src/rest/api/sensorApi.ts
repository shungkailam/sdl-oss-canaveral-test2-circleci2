import { DocType } from '../model/baseModel';
import { Sensor } from '../model/sensor';

import {
  registryService,
  REG_KEY_SOCKET_IO_CLIENT,
} from '../services/registry.service';

import {
  ReportSensorsRequest,
  ReportSensorsResponse,
  emit,
} from '../msg/index';

import { CreateDocumentResponse } from '../model/baseModel';
import { getDBService } from '../db-configurator/dbConfigurator';
import { logger } from '../util/logger';

export async function getAllSensorsForEdge(
  tenantId,
  edgeId
): Promise<Sensor[]> {
  return getDBService().getAllDocumentsForEdge<Sensor>(
    tenantId,
    edgeId,
    DocType.Sensor
  );
}

export async function getAllSensors(tenantId): Promise<Sensor[]> {
  return getDBService().getAllDocuments<Sensor>(tenantId, DocType.Sensor);
}

export async function addSensor(
  sensor: Sensor
): Promise<CreateDocumentResponse> {
  logger.info('>>> sensorApi.addSensor { sensor=', sensor);

  const socket = registryService.get(REG_KEY_SOCKET_IO_CLIENT);
  if (socket) {
    await emit<ReportSensorsRequest, ReportSensorsResponse>(
      socket,
      'reportSensors'
    )({ sensors: [sensor], tenantId: sensor.tenantId });
  }

  return getDBService().createDocument(sensor.tenantId, DocType.Sensor, sensor);
}
