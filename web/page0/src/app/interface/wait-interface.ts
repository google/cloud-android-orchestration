import {DeviceSetting} from './device-interface';

export type Wait =
  | HostCreateWait
  | HostDeleteWait
  | EnvCreateWait
  | EnvAutoHostCreateWait;

export interface HostCreateWait {
  waitUrl: string;
  metadata: {
    type: 'host-create';
    zone: string;
    runtimeAlias: string;
  };
}

export interface HostDeleteWait {
  waitUrl: string;
  metadata: {
    type: 'host-delete';
    hostUrl: string;
  };
}

export interface EnvCreateWait {
  waitUrl: string;
  metadata: {
    type: 'env-create';
    hostUrl: string;
    groupName: string;
    runtimeAlias: string;
    devices: DeviceSetting[];
  };
}

export interface EnvAutoHostCreateWait {
  waitUrl: string;
  metadata: {
    type: 'env-auto-host-create';
    zone: string;
    groupName: string;
    runtimeAlias: string;
    devices: DeviceSetting[];
  };
}

export function isHostCreateWait(wait: Wait): wait is HostCreateWait {
  return (wait as HostCreateWait).metadata.type === 'host-create';
}

export function isHostDeleteWait(wait: Wait): wait is HostDeleteWait {
  return (wait as HostDeleteWait).metadata.type === 'host-delete';
}

export function isEnvCreateWait(wait: Wait): wait is EnvCreateWait {
  return (wait as EnvCreateWait).metadata.type === 'env-create';
}

export function isEnvAutoHostCreateWait(
  wait: Wait
): wait is EnvAutoHostCreateWait {
  return (
    (wait as EnvAutoHostCreateWait).metadata.type === 'env-auto-host-create'
  );
}
