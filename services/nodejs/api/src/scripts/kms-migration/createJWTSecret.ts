import { AwsKmsKeyService } from './keyService';

const USAGE = `node createJWTSecret.js <kms master key arn>`;
async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  const keyId = process.argv[2];

  const kmsService = new AwsKmsKeyService();
  const secret = await kmsService.genJWTSecret(keyId);
  console.log(secret);
}

main();
