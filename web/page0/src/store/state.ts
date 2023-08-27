import {Environment} from 'src/app/env-interface';
import {Host} from 'src/app/host-interface';
import {Runtime, RuntimeViewStatus} from 'src/app/runtime-interface';

export interface AppState {
  runtimes: Runtime[];
  runtimesLoadStatus: RuntimeViewStatus;
  startingEnvs: Environment[];
  stoppingEnvs: Environment[];
}

export const initialState: AppState = {
  runtimes: [],
  runtimesLoadStatus: RuntimeViewStatus.initializing,
  startingEnvs: [],
  stoppingEnvs: [],
};
