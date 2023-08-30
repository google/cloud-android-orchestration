export enum HostStatus {
  starting = 'starting',
  running = 'running',
  stopping = 'stopping',
  error = 'error',
  loading = 'loading',
}

export interface Host {
  name: string;
  zone?: string;
  url?: string;
  runtime: string;
  status: HostStatus;
}
