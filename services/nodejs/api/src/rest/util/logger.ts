import * as pino from 'pino';

export const logger = pino();

export function setLogLevel(logLevel: string | number) {
  switch (logLevel) {
    case '1':
    case 1:
    case 'fatal':
      logger.level = 'fatal';
      break;
    case '2':
    case 2:
    case 'error':
      logger.level = 'error';
      break;
    case '3':
    case 3:
    case 'warn':
      logger.level = 'warn';
      break;
    case '4':
    case 4:
    case 'info':
      logger.level = 'info';
      break;
    case '5':
    case 5:
    case 'debug':
      logger.level = 'debug';
      break;
    case '6':
    case 6:
    case 'trace':
      logger.level = 'trace';
      break;
    default:
      break;
  }
}
