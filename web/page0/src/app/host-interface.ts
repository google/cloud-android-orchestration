import { Group } from './host-orchestrator.dto';

export interface Host {
  name: string;
  zone?: string;
  url: string;
  runtime: string;
  groups: Group[];
}
