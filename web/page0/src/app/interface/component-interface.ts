import {Environment} from './env-interface';
import {HostStatus} from './host-interface';
import {RuntimeStatus} from './runtime-interface';

export interface HostItem {
  name: string;
  zone?: string;
  url?: string;
  runtime: string;
  status: HostStatus;
  envs: Environment[];
}

export interface RuntimeCard {
  alias: string;
  type?: 'local' | 'on-premise' | 'cloud';
  url: string;
  hosts: HostItem[];
  status: RuntimeStatus;
}

export type EnvCard = Environment;
