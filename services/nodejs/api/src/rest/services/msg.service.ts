import {
  registryService,
  REG_KEY_SOCKET_IO,
  REG_KEY_SOCKET_IO_CLIENT,
} from './registry.service';
import { emit, emitToTenant, MESSAGE_KEY } from '../msg/index';
import { getSocketKey } from '../util/msgUtil';
import { MessageEmitter } from '../msg/base';

/**
 * Message service to simplify emit message to edge.
 */
export interface MessageService {
  /**
   * Get emitter to send message to edge.
   * @param {string} tenantId
   * @param {string} edgeId
   * @param {MESSAGE_KEY} msg
   * @returns {MessageEmitter<Q, R>} May be null if not found
   */
  getEmitterToEdge<Q, R>(
    tenantId: string,
    edgeId: string,
    msg: MESSAGE_KEY
  ): MessageEmitter<Q, R>;

  /**
   * Get emitter to send message to all edges for the tenant.
   * @param {string} tenantId
   * @param {MESSAGE_KEY} msg
   * @returns {MessageEmitter<Q, R>} May be null if not found
   */
  getEmitterToTenant<Q, R>(
    tenantId: string,
    msg: MESSAGE_KEY
  ): MessageEmitter<Q, R>;

  /**
   * Get emitter to send message to cloudmgmt server.
   * @param {MESSAGE_KEY} msg
   * @returns {MessageEmitter<Q, R>} May be null if not found
   */
  getEmitterToServer<Q, R>(msg: MESSAGE_KEY): MessageEmitter<Q, R>;
}

class MessageServiceImpl implements MessageService {
  public getEmitterToEdge<Q, R>(
    tenantId: string,
    edgeId: string,
    msg: MESSAGE_KEY
  ): MessageEmitter<Q, R> {
    const socket = registryService.get(getSocketKey(tenantId, edgeId));
    if (socket) {
      return emit<Q, R>(socket, msg);
    }
    return null;
  }

  public getEmitterToTenant<Q, R>(
    tenantId: string,
    msg: MESSAGE_KEY
  ): MessageEmitter<Q, R> {
    const socket = registryService.get(REG_KEY_SOCKET_IO);
    if (socket) {
      return emitToTenant<Q, R>(socket, tenantId, msg);
    }
    return null;
  }

  public getEmitterToServer<Q, R>(msg: MESSAGE_KEY): MessageEmitter<Q, R> {
    const socket = registryService.get(REG_KEY_SOCKET_IO_CLIENT);
    if (socket) {
      return emit<Q, R>(socket, msg);
    }
    return null;
  }
}

export const messageService: MessageService = new MessageServiceImpl();
