import * as AWS from 'aws-sdk';

export interface LogService {
  getUploadUrl(bucket: string, key: string): Promise<string>;

  getDownloadUrl(bucket: string, key: string): Promise<string>;

  deleteObject(bucket: string, key: string): Promise<boolean>;
}

const AWS_REGION = process.env.AWS_REGION || 'us-west-2';
const SIGNED_URL_EXPIRY_SECS = 600;

class AwsLogService implements LogService {
  private s3: any = null;

  constructor() {
    AWS.config.update(<any>{
      region: AWS_REGION,
    });
    this.s3 = new AWS.S3({ apiVersion: '2014-11-01' });
  }

  public async getUploadUrl(bucket: string, key: string): Promise<string> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const params = {
          Bucket: bucket,
          Key: key,
          Expires: SIGNED_URL_EXPIRY_SECS,
          ACL: 'bucket-owner-full-control',
          ContentType: 'application/x-gzip',
        };
        const signedUrl = this.s3.getSignedUrl('putObject', params);
        resolve(signedUrl);
      } catch (err) {
        reject(err);
      }
    });
  }

  public getDownloadUrl(bucket: string, key: string): Promise<string> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const params = {
          Bucket: bucket,
          Prefix: key,
          MaxKeys: 1,
        };
        const data = this.s3.listObjectsV2(params);
        if (data == null) {
          resolve(null);
        } else {
          const params = {
            Bucket: bucket,
            Key: key,
            Expires: SIGNED_URL_EXPIRY_SECS,
          };
          resolve(this.s3.getSignedUrl('getObject', params));
        }
      } catch (err) {
        reject(err);
      }
    });
  }

  public async deleteObject(bucket: string, key: string): Promise<boolean> {
    return new Promise<boolean>(async (resolve, reject) => {
      try {
        const params = {
          Bucket: bucket,
          Prefix: key,
        };
        this.s3.deleteObject(params);
        resolve(true);
      } catch (err) {
        reject(err);
      }
    });
  }
}

const logService: LogService = new AwsLogService();

export default logService;
