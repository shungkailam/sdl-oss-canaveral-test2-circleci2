// Utility for ExpressJS
import { logger } from './logger';

export function logErrors(err, req, res, next) {
  // for REST API call, report error in json response, don't send HTML
  if (req.url.indexOf('/v1/') === 0) {
    res.status(err.status || 500).send(err);
  }
  logger.error('logErrors:', err);
  next(err);
}
