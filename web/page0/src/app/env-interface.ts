export enum EnvStatus {
  starting = 'starting',
  running = 'running',
  stopping = 'stopping',
  error = 'error',
}

export interface Envrionment {
  runtimeAlias: string;
  hostUrl: string;
  groupName: string;
  devices: string[];
  status: EnvStatus;
}
