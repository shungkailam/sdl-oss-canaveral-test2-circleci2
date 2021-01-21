export function getSocketKey(tenantId: string, edgeId: string): string {
  var socketKey = [tenantId, edgeId].join('/');
  return `socket.${socketKey}`;
}
