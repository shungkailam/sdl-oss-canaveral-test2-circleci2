import { EdgeBaseModel } from './baseModel';

export interface EdgeInfo extends EdgeBaseModel {
  //
  // NumCpu
  //
  NumCPU?: string;
  //
  // TotalMemoryKB
  //
  TotalMemoryKB?: string;
  //
  // TotalStorageKB
  //
  TotalStorageKB?: string;
  //
  // GPUInfo
  //
  GPUInfo?: string;
  //
  // CPUUsage
  //
  CPUUsage?: string;
  //
  // MemoryFreeKB
  //
  MemoryFreeKB?: string;
  //
  // StorageFreeKB
  //
  StorageFreeKB?: string;
  //
  // GPUUsage
  //
  GPUUsage?: string;
}
