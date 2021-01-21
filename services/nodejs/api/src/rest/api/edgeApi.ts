import { DocType } from '../model/baseModel';
import { Edge } from '../model/edge';
import { EdgeCert } from '../model/edgeCert';
import { registryService } from '../services/registry.service';
import {
  UpdateDocumentResponse,
  CreateDocumentResponse,
} from '../model/baseModel';
import { getSocketKey } from '../util/msgUtil';
import { getDBService } from '../db-configurator/dbConfigurator';
import { logger } from '../util/logger';
import platformService from '../services/platform.service';

export async function getAllEdges(tenantId): Promise<Edge[]> {
  const edges: Edge[] = await getDBService().getAllDocuments<Edge>(
    tenantId,
    DocType.Edge
  );
  edges.forEach(e => {
    e.connected = Boolean(registryService.get(getSocketKey(e.tenantId, e.id)));
  });
  return edges;
}

export async function getAllEdgeCerts(tenantId: string): Promise<EdgeCert[]> {
  return getDBService().getAllDocuments<EdgeCert>(tenantId, DocType.EdgeCert);
}

export async function addEdge(edge: Edge): Promise<CreateDocumentResponse> {
  logger.info('>>> edgeApi.addEdge { edge=', edge);
  return getDBService().createDocument(edge.tenantId, DocType.Edge, edge);
}

export async function updateEdge(edge: Edge): Promise<UpdateDocumentResponse> {
  logger.info('>>> edgeApi.updateEdge { edge=', edge);
  return getDBService().updateDocument(
    edge.tenantId,
    edge.id,
    DocType.Edge,
    edge
  );
}

export async function createEdgeToken(edge: Edge) {
  const { tenantId, id } = edge;
  return createEdgeToken2(tenantId, id);
}

export async function createEdgeToken2(tenantId, edgeId) {
  const payload = {
    tenantId,
    edgeId,
    specialRole: 'edge',
    roles: [],
    scopes: [],
  };
  const token = await platformService.getKeyService().jwtSign(payload);
  return { token };
}
