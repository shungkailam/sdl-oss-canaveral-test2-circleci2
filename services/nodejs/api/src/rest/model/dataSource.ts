import { EdgeBaseModel, EdgeBaseModelKeys } from './baseModel';
import { CategoryInfo, CategoryInfoKeys } from './category';

/**
 * A field represents a piece of information within a mqtt payload.
 * A mqtt topic may contain multiple fields.
 * User defines fields extractable from a mqtt topic
 * by specifying DataSourceFieldInfo for each field of a DataSource in the UI.
 * The fieldType for a field together with the sensorModel
 * of DataSource are used to extract the field value from the mqtt payload.
 */
export interface DataSourceFieldInfo {
  /**
   * name of the field
   * This should be unique within the DataSource instance.
   */
  name: string;
  /**
   * mqttTopic for the field
   * For .NEXT Nice this uniquely identifies the corresponding sensor
   * for the mqtt topic.
   */
  mqttTopic: string;
  /**
   * data type for the field
   * E.g., Temperature, Presure, Custom, etc.
   * Custom means the entire mqtt payload, no special extraction will be performed.
   * (This will limit the intelligent operations Sherlock may automatically
   * perform for more specific field types.)
   * Post Nice we may allow user to provide custom extraction functions
   * for each field.
   * DataSource dataType is derived from fieldType of all fields in the DataSource.
   */
  fieldType: string;
}
export const DataSourceFieldInfoKeys = ['name', 'mqttTopic', 'fieldType'];

/**
 * A DataSourceFieldSelector is a choice of a value from a category,
 * together with a specification of scope on which fields
 * this value applies to.
 * User annotates each DataSource field with one or more
 * CategoryInfo objects.
 * Categories are the primary mechanism Sherlock provides
 * user to specify the input of a DataStream.
 * The list of categories spec is checked against each DataSource
 * to determine whether any field in the DataSource should be
 * included in the input of the DataStream.
 */
export interface DataSourceFieldSelector extends CategoryInfo {
  /**
   * Name of the fields this CategoryInfo is applicable to.
   * The special value '__ALL__' means this CategoryInfo is applicable to
   * all fields in this DataSource.
   */
  scope: string[];
}
export const DataSourceFieldSelectorKeys = ['scope'].concat(CategoryInfoKeys);

/**
 * A DataSource represents a logical IoT Sensor or Gateway.
 * Note: this is merely a grouping construct to hold meta info
 * for sensors. The act of defining DataSource does not cause
 * the mqtt message to flow into Kafka. To do that, one must
 * 'tap' into the DataSource by creating DataStreams.
 */
export interface DataSource extends EdgeBaseModel {
  /**
   * name of the DataSource
   */
  name: string;
  /**
   * type of the DataSource.
   * Either Sensor or Gateway
   */
  type: 'Sensor' | 'Gateway';
  /**
   * Sensor model
   * This is specific to .NEXT Nice.
   * Since we can't currently detect sensor capability,
   * we will have a list of supported sensorModel values
   * which maps to predefined sensor payload format.
   */
  sensorModel: string;
  /**
   * Sensor connection type.
   * Either Secure or Unsecure
   */
  connection: 'Secure' | 'Unsecure';
  /**
   * User defined fields to extract data from the mqtt payload.
   */
  fields: DataSourceFieldInfo[];
  /**
   * A list of DataSourceFieldSelector user assigned to the DataSource
   * to allow user to use Category selectors to identify
   * source to a DataStream.
   * Selectors with different category id are combined with AND,
   * while selectors with the same category id are combined with OR.
   */
  selectors: DataSourceFieldSelector[];
  /**
   * Sensor protocol
   */
  protocol: 'MQTT' | 'RTSP' | 'GIGEVISION' | 'OTHER';
  /**
   * Type of authentication used by sensor
   */
  authType: 'CERTIFICATE' | 'PASSWORD' | 'TOKEN';
  // note: Sensor dataType is a derived property
  // from the collection of sensor fields.
  // e.g., if there is only one field, then dataType = field.fieldType
  // more generally, dataType = union of field types
}
export const DataSourceKeys = [
  'name',
  'type',
  'sensorModel',
  'connection',
  'fields',
  'selectors',
  'protocol',
  'authType',
].concat(EdgeBaseModelKeys);
