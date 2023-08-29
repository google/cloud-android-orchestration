import {RuntimeViewStatus} from 'src/app/interface/runtime-interface';
import {runtimeToEnvList} from 'src/app/interface/utils';
import {Wait} from '../interface/wait-interface';
import {
  Action,
  EnvCreateStartAction,
  EnvDeleteStartAction,
  HostCreateCompleteAction,
  HostCreateStartAction,
  InitAction,
  RuntimeLoadAction,
  RuntimeLoadCompleteAction,
  RuntimeRefreshStartAction,
  RuntimeRegisterCompleteAction,
  RuntimeRegisterErrorAction,
  RuntimeRegisterStartAction,
  RuntimeUnregisterAction,
} from './actions';
import {AppState, initialState} from './state';

type ActionType = string;
type Reducer = (action: any) => (prevState: AppState) => AppState;

const identityReducer = (action: Action) => (prevState: AppState) => prevState;

const reducers: {[key: ActionType]: Reducer} = {
  init: (action: InitAction) => (prevState: AppState) => initialState,
  'runtime-refresh-start':
    (action: RuntimeRefreshStartAction) => prevState => ({
      ...prevState,
      runtimesLoadStatus: RuntimeViewStatus.refreshing,
      runtimes: [],
    }),

  'runtime-load': (action: RuntimeLoadAction) => prevState => {
    const envs = runtimeToEnvList(action.runtime);

    const nextStartingEnvs = prevState.startingEnvs.filter(
      startingEnv =>
        startingEnv.runtimeAlias !== action.runtime.alias ||
        !envs.find(
          env =>
            env.groupName === startingEnv.groupName &&
            env.hostUrl === startingEnv.hostUrl
        )
    );

    const nextStoppingEnvs = prevState.stoppingEnvs.filter(
      stoppingEnv =>
        stoppingEnv.runtimeAlias !== action.runtime.alias ||
        !!envs.find(
          env =>
            env.groupName === stoppingEnv.groupName &&
            env.hostUrl === stoppingEnv.hostUrl
        )
    );

    return {
      ...prevState,
      stoppingEnvs: nextStoppingEnvs,
      startingEnvs: nextStartingEnvs,
      runtimes: [...prevState.runtimes, action.runtime],
    };
  },

  'runtime-load-complete': (action: RuntimeLoadCompleteAction) => prevState => {
    return {
      ...prevState,
      runtimesLoadStatus: RuntimeViewStatus.done,
    };
  },

  'runtime-register-start':
    (action: RuntimeRegisterStartAction) => prevState => {
      return {
        ...prevState,
        runtimesLoadStatus: RuntimeViewStatus.registering,
      };
    },

  'runtime-register-complete':
    (action: RuntimeRegisterCompleteAction) => prevState => {
      return {
        ...prevState,
        runtimesLoadStatus: RuntimeViewStatus.done,
        runtimes: [...prevState.runtimes, action.runtime],
      };
    },

  'runtime-register-error':
    (action: RuntimeRegisterErrorAction) => prevState => {
      return {
        ...prevState,
        runtimesLoadStatus: RuntimeViewStatus.register_error,
      };
    },

  'runtime-unregister': (action: RuntimeUnregisterAction) => prevState => {
    return {
      ...prevState,
      runtimes: prevState.runtimes.filter(item => item.alias !== action.alias),
    };
  },

  // TODO: long polling
  'env-create-start': (action: EnvCreateStartAction) => prevState => {
    return {
      ...prevState,
      startingEnvs: [...prevState.startingEnvs, action.env],
    };
  },

  'env-delete-start': (action: EnvDeleteStartAction) => prevState => {
    return {
      ...prevState,
      stoppingEnvs: [...prevState.stoppingEnvs, action.target],
    };
  },

  'host-create-start': (action: HostCreateStartAction) => prevState => {
    return {
      ...prevState,
      waits: {...prevState.waits, [action.wait.waitUrl]: action.wait},
    };
  },

  'host-create-complete': (action: HostCreateCompleteAction) => prevState => {
    const newHost = action.host;
    return {
      ...prevState,
      runtimes: prevState.runtimes.map(runtime => {
        if (runtime.alias !== newHost.runtime) {
          return runtime;
        }

        if (
          runtime.hosts.find(
            host => host.name === newHost.name && host.zone === newHost.zone
          )
        ) {
          return runtime;
        }

        return {
          ...runtime,
          hosts: [...runtime.hosts, newHost],
        };
      }),
      waits: Object.keys(prevState.waits)
        .filter(key => key !== action.waitUrl)
        .reduce((obj: {[key: string]: Wait}, key) => {
          obj[key] = prevState.waits[key];
          return obj;
        }, {}),
    };
  },
} as const;

const handlers: Map<ActionType, Reducer> = new Map<ActionType, Reducer>(
  Object.entries(reducers).map(([actionType, reducer]) => {
    return [actionType as ActionType, reducer as Reducer];
  })
);

export function match(action: Action) {
  const reducer = handlers.get(action.type);
  if (reducer) {
    return reducer(action);
  }

  console.error('No reducer registered for action type ', action.type);
  return identityReducer(action);
}
