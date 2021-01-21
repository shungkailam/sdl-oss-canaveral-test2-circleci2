import { expect } from 'chai';
import 'mocha';
import { setLogLevel, logger } from './logger';

describe('setLogLevel', () => {
  it('should update logger.level', () => {
    setLogLevel(5);
    expect(logger.level).to.equal('debug');

    setLogLevel('warn');
    expect(logger.level).to.equal('warn');

    setLogLevel('5');
    expect(logger.level).to.equal('debug');

    setLogLevel('1');
    expect(logger.level).to.equal('fatal');

    setLogLevel('debug');
    expect(logger.level).to.equal('debug');
  });
});
