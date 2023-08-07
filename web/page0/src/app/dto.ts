export interface HostCreateDto {
  machine_type: string;
  min_cpu_platform: string;
}

export interface HostResponseDto {
  name: string;
  zone?: string;
  groups: string[];
}
