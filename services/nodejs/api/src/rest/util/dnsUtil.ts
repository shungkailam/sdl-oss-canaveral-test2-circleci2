const dns = require('dns');

export function dnsLookup(name, deflt) {
  return new Promise((resolve, reject) => {
    dns.lookup(name, (err, address, family) => {
      // console.log('address: %j family: IPv%s', address, family);
      if (err) {
        if (deflt) {
          resolve(deflt);
        } else {
          reject(err);
        }
      } else {
        resolve(address);
      }
    });
  });
}
