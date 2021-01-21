//
// Script to stress memory usage
// Useful in testing memory based K8s HPA
//
const USAGE = `\nUsage: node stressMemory.js <size> <iteration> <sleep ms>\n`;
const start = Date.now();

function getET() {
  const now = Date.now();
  const ms = Math.round(now - start);
  return `${ms} ms`;
}
async function main() {
  if (process.argv.length < 5) {
    console.log(USAGE);
    process.exit(1);
  }
  const size = parseInt(process.argv[2], 10);
  const iter = parseInt(process.argv[3], 10);
  const sleepMS = parseInt(process.argv[4], 10);
  console.log(`>>> stress memory, size=${size}, iteration=${iter}`);
  const arr = Array(size);

  for (let i = 0; i < iter; i++) {
    for (let j = 0; j < size; j++) {
      arr[j] = Math.random();
    }
    console.log(`${getET()}: iteration ${i}`);
  }
  console.log(`>>> sleeping for ${sleepMS} ms`);
  await new Promise((resolve, reject) => {
    setTimeout(() => {
      resolve();
    }, sleepMS);
  });
}

main();
