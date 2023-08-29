export interface Wait {
  //   requestUrl: string;
  //   operationName: string;
  waitUrl: string;
  metadata: WaitMetadata;
}

type WaitMetadata = HostCreateWaitMetadata | HostDeleteWaitMetaData;

export interface HostCreateWaitMetadata {
  type: 'host-create';
  zone: string;
  runtimeAlias: string;
}

interface HostDeleteWaitMetaData {
  type: 'host-delete';
  zone: string;
  hostName: string;
}
