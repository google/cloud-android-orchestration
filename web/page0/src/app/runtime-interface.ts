import { Host } from './host-interface';

export enum RuntimeStatus {
  valid = 'valid',
  error = 'error',
  loading = 'loading',
}

export interface Runtime {
  alias: string;
  type?: 'local' | 'on-premise' | 'cloud';
  url: string;
  zones?: string[];
  hosts: Host[];
  status: RuntimeStatus;
}

export enum RuntimeViewStatus {
  initializing = 'initializing',
  refreshing = 'refreshing',
  registering = 'registering',
  register_error = 'register_error',
  done = 'done',
}
