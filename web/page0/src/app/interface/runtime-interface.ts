export enum RuntimeStatus {
  valid = 'valid',
  error = 'error',
  loading = 'loading',
}

export enum RuntimeType {
  local = 'local',
  onPremise = 'on-premise',
  cloud = 'cloud',
}

export interface RuntimeInfo {
  type: RuntimeType;
}

export interface Runtime {
  alias: string;
  type?: RuntimeType;
  url: string;
  zones?: string[];
  status: RuntimeStatus;
}

export enum RuntimeViewStatus {
  initializing = 'initializing',
  refreshing = 'refreshing',
  registering = 'registering',
  register_error = 'register_error',
  done = 'done',
}
