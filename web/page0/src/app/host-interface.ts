export interface Host {
  name: string;
  url: string;
  zone?: string;
  runtime: string;
  groups: string[];
}

export interface HostInfo {
  name: string;
  url: string;
  zone?: string;
  groups: string[];
}
