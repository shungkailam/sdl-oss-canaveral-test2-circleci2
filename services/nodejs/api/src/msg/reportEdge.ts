import { ReportEdge, ReportEdgeRequest } from '../rest/msg/index';

import { registryService } from '../rest/services/registry.service';
import { getSocketKey } from '../rest/util/msgUtil';
import { getDBService } from '../rest/db-configurator/dbConfigurator';
import { DocType } from '../rest/model/baseModel';
import { Edge } from '../rest/model/edge';
import { logger } from '../rest/util/logger';

// ** Activate edge step is dropped at least for .NEXT Nice **
// along with edgeStatus

// Edge will periodically report itself to the cloudmgmt,
// including right after boot up.
// As part of edge configuration (before connecting to cloud)
// it will be assigned tenantId and edgeId.
// After edge report itself on boot up, it should also report its sensors.
export const reportEdge: ReportEdge = async (
  socket: SocketIO.Socket,
  req: ReportEdgeRequest
) => {
  try {
    logger.info('>>> edge reported: ', req);
    const reqDoc = req.doc;

    // TODO FIXME - edge currently does not have
    // all edge metadata from config server
    // so do not update edge here
    // const resp = await getDBService().createDocument(
    //   req.tenantId,
    //   DocType.Edge,
    //   reqDoc
    // );
    // const docId = resp._id;
    const docId = reqDoc.id;
    if (!docId || !req.tenantId) {
      // docId and tenantId are required
      throw Error('reportEdge: Bad input!');
    }

    // when an Edge report itself to cloudmgmt,
    // we register its socket in our registry
    // so we can later retrieve it based on edgeId
    // TODO: should socket key contain both edgeId and tenantId?
    registryService.register(getSocketKey(req.tenantId, docId), socket);

    // have the socket join the tenant id room
    socket.join(req.tenantId);

    const doc = await getDBService().getDocument<Edge>(
      req.tenantId,
      docId,
      DocType.Edge
    );
    return {
      doc,
      statusCode: 200,
    };
  } catch (e) {
    logger.error('>>> reportEdge: caught exception:', e);
    return {
      doc: null,
      statusCode: 500,
    };
  }

  // TODO: handle the case where server may have created edge successfully,
  // TODO: but failed to notify client. In this case, DB will have an edge
  // TODO: instance with matching serial number, but this request will not have edgeId
};
