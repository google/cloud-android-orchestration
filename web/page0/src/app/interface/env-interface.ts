import {DeviceSetting} from './device-interface';

export enum EnvStatus {
  starting = 'starting',
  running = 'running',
  stopping = 'stopping',
  error = 'error',
}

export interface Environment {
  runtimeAlias: string;
  hostUrl: string;
  groupName: string;
  devices: DeviceSetting[];
  status: EnvStatus;
}

export interface CommonEnvConfig {
  group_name: string;
}

export interface DiskConfig {
  default_build: string;
}

export interface InstanceConfig {
  name: string;
  disk: DiskConfig;
}

export interface EnvConfig {
  common: CommonEnvConfig;
  instances: InstanceConfig[];
}
