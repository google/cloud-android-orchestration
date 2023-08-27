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
  | RuntimeRegisterStartAction;

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
