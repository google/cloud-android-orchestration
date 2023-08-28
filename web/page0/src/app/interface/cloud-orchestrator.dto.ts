// Should be aligned with api/v1/instancemanager.go

export interface CreateHostRequest {
  host_instance: HostInstance;
}

export interface HostInstance {
  name?: string;
  boot_disk_size_gb?: number;
  gcp?: GCPInstance;
}
export interface GCPInstance {
  machine_type: string;
  min_cpu_platform: string;
}

export interface Operation {
  name: string;
  metadata?: any;
  done: boolean;
}

export interface OperationResult {
  error?: object;
  response?: string;
}

export interface ListHostsResponse {
  items?: HostInstance[];
  nextPageToken?: string;
}

export interface RuntimeInfo {
  type: 'local' | 'on-premise' | 'cloud';
  // TODO: Add other information e.g. chipset, machine_type
}

export interface Zone {
  name: string;
}

export interface ListZonesResponse {
  items?: Zone[];
}
