import {Environment} from 'src/app/interface/env-interface';
import {Runtime, RuntimeViewStatus} from 'src/app/interface/runtime-interface';
import {Wait} from 'src/app/interface/wait-interface';
import {Host} from '../interface/host-interface';

export interface AppState {
  runtimes: Runtime[];
  hosts: Host[];
  envs: Environment[];
  runtimesLoadStatus: RuntimeViewStatus;
  waits: {[key: string]: Wait};
}

export const initialState: AppState = {
  runtimes: [],
  hosts: [],
  envs: [],
  runtimesLoadStatus: RuntimeViewStatus.initializing,
  waits: {},
};
