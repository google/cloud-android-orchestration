import {Environment} from 'src/app/interface/env-interface';
import {Runtime} from 'src/app/interface/runtime-interface';
import {Host} from 'src/app/interface/host-interface';
import {Wait} from 'src/app/interface/wait-interface';

export type Action =
  | InitAction
  | RefreshStartAction
  | RefreshCompleteAction
  | RuntimeRegisterCompleteAction
  | RuntimeUnregisterAction
  | RuntimeInitializeAction
  | RuntimeLoadAction
  | RuntimeRegisterErrorAction
  | RuntimeRegisterStartAction
  | HostCreateStartAction
  | HostCreateCompleteAction
  | HostCreateErrorAction
  | HostDeleteStartAction
  | HostDeleteCompleteAction
  | HostDeleteErrorAction
  | HostLoadAction
  | EnvLoadAction
  | EnvCreateStartAction
  | EnvCreateErrorAction
  | EnvCreateCompleteAction
  | EnvAutoHostCreateStartAction
  | EnvAutoHostCreateCompleteAction;

export interface RefreshStartAction {
  type: 'refresh-start';
}

export interface RefreshCompleteAction {
  type: 'refresh-complete';
}

export interface InitAction {
  type: 'init';
}

export interface RuntimeRegisterStartAction {
  type: 'runtime-register-start';
}

export interface RuntimeRegisterCompleteAction {
  type: 'runtime-register-complete';
  runtime: Runtime;
}

export interface RuntimeRegisterErrorAction {
  type: 'runtime-register-error';
}

export interface RuntimeUnregisterAction {
  type: 'runtime-unregister';
  alias: string;
}

export interface RuntimeInitializeAction {
  type: 'runtime-init';
}

export interface RuntimeLoadAction {
  type: 'runtime-load';
  runtime: Runtime;
}

export interface HostLoadAction {
  type: 'host-load';
  host: Host;
}

export interface EnvLoadAction {
  type: 'env-load';
  env: Environment;
}

export interface RuntimeLoadCompleteAction {
  type: 'runtime-load-complete';
}

export interface HostCreateStartAction {
  type: 'host-create-start';
  wait: Wait;
}

export interface HostCreateCompleteAction {
  type: 'host-create-complete';
  waitUrl: string;
  host: Host;
}

export interface HostCreateErrorAction {
  type: 'host-create-error';
  waitUrl?: string;
}

export interface HostDeleteStartAction {
  type: 'host-delete-start';
  wait: Wait;
}

export interface HostDeleteCompleteAction {
  type: 'host-delete-complete';
  waitUrl: string;
}

export interface HostDeleteErrorAction {
  type: 'host-delete-error';
  waitUrl: string;
}

export interface EnvCreateStartAction {
  type: 'env-create-start';
  wait: Wait;
}

export interface EnvCreateCompleteAction {
  type: 'env-create-complete';
  waitUrl: string;
  env: Environment;
}

export interface EnvCreateErrorAction {
  type: 'env-create-error';
  waitUrl?: string;
}

export interface EnvAutoHostCreateStartAction {
  type: 'env-auto-host-create-start';
  wait: Wait;
}

export interface EnvAutoHostCreateCompleteAction {
  type: 'env-auto-host-create-complete';
  waitUrl: string;
  host: Host;
}
