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

export interface EnvConfig {
  common: {
    group_name: string;
  };

  instances: {
    name: string;
    disk: {
      default_build: string;
    };
  }[];
}
