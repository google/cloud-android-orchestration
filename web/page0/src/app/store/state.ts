import {Environment} from 'src/app/interface/env-interface';
import {Runtime, RuntimeViewStatus} from 'src/app/interface/runtime-interface';
import {Wait} from 'src/app/interface/wait-interface';

export interface AppState {
  runtimes: Runtime[];
  runtimesLoadStatus: RuntimeViewStatus;
  startingEnvs: Environment[];
  stoppingEnvs: Environment[];
  waits: {[key: string]: Wait};
}

export const initialState: AppState = {
  runtimes: [],
  runtimesLoadStatus: RuntimeViewStatus.initializing,
  startingEnvs: [],
  stoppingEnvs: [],
  waits: {},
};
