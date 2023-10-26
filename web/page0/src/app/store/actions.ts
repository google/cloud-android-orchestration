import {Environment} from 'src/app/interface/env-interface';
import {Runtime} from 'src/app/interface/runtime-interface';
import {Host} from 'src/app/interface/host-interface';
import {Wait} from 'src/app/interface/wait-interface';

export enum ActionType {
  Init = 'init',
  RefreshStart = 'refresh-start',
  RefreshComplete = 'refresh-complete',
  RuntimeRegisterComplete = 'runtime-register-complete',
  RuntimeUnregister = 'runtime-unregister',
  RuntimeInitialize = 'runtime-initialize',
  RuntimeLoad = 'runtime-load',
  RuntimeLoadComplete = 'runtime-load-complete',
  RuntimeRegisterError = 'runtime-register-error',
  RuntimeRegisterStart = 'runtime-register-start',
  HostCreateStart = 'host-create-start',
  HostCreateComplete = 'host-create-complete',
  HostCreateError = 'host-create-error',
  HostDeleteStart = 'host-delete-start',
  HostDeleteComplete = 'host-delete-complete',
  HostDeleteError = 'host-delete-error',
  HostLoad = 'host-load',
  EnvLoad = 'env-load',
  EnvCreateStart = 'env-create-start',
  EnvCreateError = 'env-create-error',
  EnvCreateComplete = 'env-create-complete',
  EnvAutoHostCreateStart = 'env-auto-host-create-start',
  EnvAutoHostCreateComplete = 'env-auto-host-create-complete',
}

export type Action =
  | InitAction
  | RefreshStartAction
  | RefreshCompleteAction
  | RuntimeRegisterCompleteAction
  | RuntimeUnregisterAction
  | RuntimeInitializeAction
  | RuntimeLoadAction
  | RuntimeLoadCompleteAction
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
  type: ActionType.RefreshStart;
}

export interface RefreshCompleteAction {
  type: ActionType.RefreshComplete;
}

export interface InitAction {
  type: ActionType.Init;
}

export interface RuntimeRegisterStartAction {
  type: ActionType.RuntimeRegisterStart;
}

export interface RuntimeRegisterCompleteAction {
  type: ActionType.RuntimeRegisterComplete;
  runtime: Runtime;
}

export interface RuntimeRegisterErrorAction {
  type: ActionType.RuntimeRegisterError;
}

export interface RuntimeUnregisterAction {
  type: ActionType.RuntimeUnregister;
  alias: string;
}

export interface RuntimeInitializeAction {
  type: ActionType.RuntimeInitialize;
}

export interface RuntimeLoadAction {
  type: ActionType.RuntimeLoad;
  runtime: Runtime;
}

export interface HostLoadAction {
  type: ActionType.HostLoad;
  host: Host;
}

export interface EnvLoadAction {
  type: ActionType.EnvLoad;
  env: Environment;
}

export interface RuntimeLoadCompleteAction {
  type: ActionType.RuntimeLoadComplete;
}

export interface HostCreateStartAction {
  type: ActionType.HostCreateStart;
  wait: Wait;
}

export interface HostCreateCompleteAction {
  type: ActionType.HostCreateComplete;
  waitUrl: string;
  host: Host;
}

export interface HostCreateErrorAction {
  type: ActionType.HostCreateError;
  waitUrl?: string;
}

export interface HostDeleteStartAction {
  type: ActionType.HostDeleteStart;
  wait: Wait;
}

export interface HostDeleteCompleteAction {
  type: ActionType.HostDeleteComplete;
  waitUrl: string;
}

export interface HostDeleteErrorAction {
  type: ActionType.HostDeleteError;
  waitUrl: string;
}

export interface EnvCreateStartAction {
  type: ActionType.EnvCreateStart;
  wait: Wait;
}

export interface EnvCreateCompleteAction {
  type: ActionType.EnvCreateComplete;
  waitUrl: string;
  env: Environment;
}

export interface EnvCreateErrorAction {
  type: ActionType.EnvCreateError;
  waitUrl?: string;
}

export interface EnvAutoHostCreateStartAction {
  type: ActionType.EnvAutoHostCreateStart;
  wait: Wait;
}

export interface EnvAutoHostCreateCompleteAction {
  type: ActionType.EnvAutoHostCreateComplete;
  waitUrl: string;
  host: Host;
}
