import { ReportSensors, ReportSensorsRequest } from '../rest/msg/index';
import { getDBService } from '../rest/db-configurator/dbConfigurator';
import { DocType } from '../rest/model/baseModel';
import { logger } from '../rest/util/logger';

export const reportSensors: ReportSensors = async (
  socket: SocketIO.Socket,
  req: ReportSensorsRequest
) => {
  try {
    logger.info('>>> sensors reported: ', req);
    // TODO: FIXME: may need to page this if sensors count is large
    const promises = req.sensors.map(sensor =>
      getDBService().createDocument(req.tenantId, DocType.Sensor, sensor)
    );
    await Promise.all(promises);
    return {
      statusCode: 201,
    };
  } catch (e) {
    logger.error('>>> reportSensors: caught exception:', e);
    return {
      statusCode: 500,
    };
  }
};
