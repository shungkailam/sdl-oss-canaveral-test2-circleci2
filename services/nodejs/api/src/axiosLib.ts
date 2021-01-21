import AxiosLib from 'axios';

const db_host = process.env.IOT_MGMT_DB_SERVICE_HOST || 'localhost';

export const axiosInstance = AxiosLib.create({
  baseURL: `http://${db_host}:9200`,
  headers: { 'Content-Type': 'application/json' },
  timeout: 1000,
});
