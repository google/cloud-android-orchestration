export type Wait = HostCreateWait | HostDeleteWait;

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

export function isHostCreateWait(wait: Wait): wait is HostCreateWait {
  return (wait as HostCreateWait).metadata.type === 'host-create';
}

export function isHostDeleteWait(wait: Wait): wait is HostDeleteWait {
  return (wait as HostDeleteWait).metadata.type === 'host-delete';
}
