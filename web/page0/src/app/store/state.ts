import {Environment} from 'src/app/interface/env-interface';
import {Runtime, RuntimeViewStatus} from 'src/app/interface/runtime-interface';

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
