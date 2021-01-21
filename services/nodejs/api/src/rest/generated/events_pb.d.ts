// package: sherlock
// file: events.proto

import * as jspb from 'google-protobuf';

export class Event extends jspb.Message {
  getUuid(): string;
  setUuid(value: string): void;

  getTimestamp(): number;
  setTimestamp(value: number): void;

  getObjectuuid(): string;
  setObjectuuid(value: string): void;

  getEventtype(): EventType;
  setEventtype(value: EventType): void;

  getObjecttype(): ObjectType;
  setObjecttype(value: ObjectType): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Event.AsObject;
  static toObject(includeInstance: boolean, msg: Event): Event.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: Event,
    writer: jspb.BinaryWriter
  ): void;
  static deserializeBinary(bytes: Uint8Array): Event;
  static deserializeBinaryFromReader(
    message: Event,
    reader: jspb.BinaryReader
  ): Event;
}

export namespace Event {
  export type AsObject = {
    uuid: string;
    timestamp: number;
    objectuuid: string;
    eventtype: EventType;
    objecttype: ObjectType;
  };
}

export enum EventType {
  ADD = 0,
  DELETE = 1,
  UPDATE = 2,
}

export enum ObjectType {
  CATEGORY = 0,
  DATASOURCE = 1,
  DATASTREAM = 2,
  SCRIPT = 3,
  SENSOR = 4,
}
