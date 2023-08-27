import {Environment} from 'src/app/env-interface';
import {Runtime} from 'src/app/runtime-interface';

export type Action =
  | InitAction
  | RuntimeRegisterCompleteAction
  | RuntimeUnregisterAction
  | RuntimeInitializeAction
  | RuntimeRefreshStartAction
  | RuntimeLoadAction
  | RuntimeLoadCompleteAction
  | RuntimeRegisterErrorAction
  | RuntimeRegisterStartAction
  | EnvCreateStartAction
  | EnvDeleteStartAction;

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

export interface RuntimeRefreshStartAction {
  type: 'runtime-refresh-start';
}

export interface RuntimeLoadAction {
  type: 'runtime-load';
  runtime: Runtime;
}

export interface RuntimeLoadCompleteAction {
  type: 'runtime-load-complete';
}

export interface EnvCreateStartAction {
  type: 'env-create-start';
  env: Environment;
}

export interface EnvDeleteStartAction {
  type: 'env-delete-start';
  target: Environment;
}
