// Should be aligned with api/v1/instancemanager.go

export declare interface CreateHostRequest {
  host_instance: HostInstance;
}

export declare interface HostInstance {
  name?: string;
  boot_disk_size_gb?: number;
  gcp?: GCPInstance;
}
export declare interface GCPInstance {
  machine_type: string;
  min_cpu_platform: string;
}

export declare interface Operation {
  name: string;
  metadata?: any;
  done: boolean;
}

export declare interface OperationResult {
  error?: object;
  response?: string;
}

export declare interface ListHostsResponse {
  items?: HostInstance[];
  nextPageToken?: string;
}

export declare interface RuntimeConfig {
  instance_manager_type: 'GCP' | 'local';
  // TODO: Add other information e.g. chipset, machine_type
}

export declare interface Zone {
  name: string;
}

export declare interface ListZonesResponse {
  items?: Zone[];
}
