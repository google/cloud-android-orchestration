import {Group} from './host-orchestrator.dto';

export enum HostStatus {
  starting = 'starting',
  running = 'running',
  stopping = 'stopping',
  error = 'error',
}

export interface Host {
  name: string;
  zone?: string;
  url?: string;
  runtime: string;
  groups: Group[];
  status: HostStatus;
}
