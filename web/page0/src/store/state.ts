import {Runtime, RuntimeViewStatus} from 'src/app/runtime-interface';

export interface AppState {
  runtimes: Runtime[];
  runtimesLoadStatus: RuntimeViewStatus;
}

export const initialState: AppState = {
  runtimes: [],
  runtimesLoadStatus: RuntimeViewStatus.initializing,
};
