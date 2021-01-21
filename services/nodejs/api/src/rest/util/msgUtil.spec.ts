import { expect } from 'chai';
import 'mocha';
import { getSocketKey } from './msgUtil';

describe('getSocketKey', () => {
  it('should return correct format', () => {
    const result = getSocketKey('foo-bar');
    expect(result).to.equal('socket.foo-bar');
  });
});
