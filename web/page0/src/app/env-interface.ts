interface BaseEnvrionment {
  runtime: string;
  group_id: string;
  devices: any[];
  created_at: number;
  favorite: boolean;
  status: 'starting' | 'stopping' | 'running' | 'error';
}

export interface RemoteEnvironment extends BaseEnvrionment {
  env_type: 'remote';
  host: string;
  expired_at: number;
}

export interface LocalEnvironment extends BaseEnvrionment {
  env_type: 'local';
}

export type Envrionment = LocalEnvironment | RemoteEnvironment;
