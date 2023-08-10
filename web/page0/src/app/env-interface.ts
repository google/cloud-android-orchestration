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
  devices: string[];
  status: EnvStatus;
}
