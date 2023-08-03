export interface Runtime {
  alias: string;
  runtime_type: 'local' | 'on-premise' | 'cloud';
  url: string;
  hosts: string[];
  status: 'valid' | 'error' | 'loading';
}

export interface RuntimeAdditionalInfo {
  runtime_type: 'local' | 'on-premise' | 'cloud';
  hosts: string[];
  status: 'valid' | 'error' | 'loading';
}
