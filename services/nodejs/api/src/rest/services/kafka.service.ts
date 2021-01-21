import { registryService, REG_KEY_KAFKA_SERVICE } from './registry.service';
import uuidv4 = require('uuid/v4');
import * as event_pb from '../generated/events_pb';
import {
  KAFKA_NOTIFICATION_TOPIC_CATEGORY,
  KAFKA_NOTIFICATION_TOPIC_DATASOURCE,
  KAFKA_NOTIFICATION_TOPIC_DATASTREAM,
  KAFKA_NOTIFICATION_TOPIC_SENSOR,
  KAFKA_NOTIFICATION_TOPIC_SCRIPT,
} from '../constants';
import { dnsLookup } from '../util/dnsUtil';
const kafka = require('kafka-node');

const Producer = kafka.Producer;
const Client = kafka.Client;

export interface KafkaService {
  /**
   * Initialize Kafka producer.
   * Currently this should only be called on Edge,
   * since only Edge currently requires Kafka notification.
   */
  initialize();

  /**
   * Whether Kafka service is ready.
   * @returns {boolean}
   */
  isReady(): boolean;

  /**
   * Use Kafka service to publish entity CRUD event.
   * Only works when isReady() === true.
   * @param {string} entityId
   * @param {ObjectType} entityType
   * @param {EventType} eventType
   * @returns {Promise<any>}
   */
  notifyEntityCrudEvent(
    entityId: string,
    entityType: event_pb.ObjectType,
    eventType: event_pb.EventType
  ): Promise<any>;
}

/**
 * Create protobuf event for publishing to Kafka.
 * @param {string} entityId
 * @param {ObjectType} entityType
 * @param {EventType} eventType
 * @returns {Event}
 */
function createEvent(
  entityId: string,
  entityType: event_pb.ObjectType,
  eventType: event_pb.EventType
) {
  const e = new event_pb.Event();
  e.setUuid(uuidv4());
  e.setObjectuuid(entityId);
  e.setTimestamp(new Date().getTime());
  e.setEventtype(eventType);
  e.setObjecttype(entityType);
  return e;
}

/**
 * Get Kafka topic name for the object type.
 * @param {ObjectType} entityType
 * @returns {string} May return null if notification is not supported for the object type.
 */
function getTopic(entityType: event_pb.ObjectType): string {
  switch (entityType) {
    case event_pb.ObjectType.CATEGORY:
      return KAFKA_NOTIFICATION_TOPIC_CATEGORY;
    case event_pb.ObjectType.DATASOURCE:
      return KAFKA_NOTIFICATION_TOPIC_DATASOURCE;
    case event_pb.ObjectType.DATASTREAM:
      return KAFKA_NOTIFICATION_TOPIC_DATASTREAM;
    case event_pb.ObjectType.SCRIPT:
      return KAFKA_NOTIFICATION_TOPIC_SCRIPT;
    case event_pb.ObjectType.SENSOR:
      return KAFKA_NOTIFICATION_TOPIC_SENSOR;
    default:
      break;
  }
  return null;
}

class KafkaServiceImpl implements KafkaService {
  private producer = null;
  private ready = false;

  // should only be called on edge
  public initialize() {
    if (this.producer) {
      return;
    }
    dnsLookup('zk-0.zk-svc.default', 'localhost').then(host => {
      const port = 2181;
      const client = new Client(`${host}:${port}`);
      const producer = new Producer(client, { requireAcks: 1 });
      // only do the following if is Edge
      producer.on('ready', () => {
        this.ready = true;
      });

      producer.on('error', function(err) {
        console.log('Error:KafkaServiceImpl:', err);
      });
      this.producer = producer;
    });
  }

  public isReady(): boolean {
    return this.ready;
  }

  public notifyEntityCrudEvent(
    entityId: string,
    entityType: event_pb.ObjectType,
    eventType: event_pb.EventType
  ): Promise<any> {
    return new Promise((resolve, reject) => {
      if (!this.producer) {
        reject(
          Error(
            `No Kafka producer: make sure you are on Edge and initialize has been called.`
          )
        );
        return;
      }
      if (!this.ready) {
        reject(Error(`Kafka producer not ready: please try again later.`));
        return;
      }
      const topic = getTopic(entityType);
      if (!topic) {
        reject(Error(`No kafka topic for object type: ${entityType}`));
        return;
      }
      const e = createEvent(entityId, entityType, eventType);
      console.log('>>> sending to Kafka protobuf event: ', e.toObject());
      const payload = {
        attributes: 0,
        messages: Buffer.from(e.serializeBinary().buffer),
        partition: 0,
        topic,
      };
      this.producer.send([payload], (err, result) => {
        if (err) {
          reject(err);
        } else {
          resolve(result);
        }
      });
    });
  }
}

export const kafkaService: KafkaService = new KafkaServiceImpl();
