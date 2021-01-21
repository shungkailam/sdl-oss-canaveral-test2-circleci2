import { EdgeBaseModel } from './baseModel';

export interface EdgeCert extends EdgeBaseModel {
  /**
   * Certificate for the edge.
   * deprecated
   */
  certificate: string;
  /**
   * Encrypted private key.
   * deprecated
   */
  privateKey: string;
  /**
   * Certificate for the edge.
   */
  clientCertificate: string;
  /**
   * Encrypted private key.
   */
  clientPrivateKey: string;
  /**
   * Certificate for the edge.
   */
  edgeCertificate: string;
  /**
   * Encrypted private key.
   */
  edgePrivateKey: string;
  /**
   * For security purpose, EdgeCert can only be
   * retrieved once during edge on-boarding.
   * After that locked will be set to true and
   * the REST API endpoint for getting EdgeCert
   * will throw error.
   */
  locked: boolean;
}
