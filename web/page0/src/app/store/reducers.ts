import {RuntimeViewStatus} from 'src/app/interface/runtime-interface';
import {isHostDeleteWait, Wait} from '../interface/wait-interface';
import {
  Action,
  EnvCreateStartAction,
  HostCreateCompleteAction,
  HostCreateStartAction,
  HostDeleteCompleteAction,
  HostDeleteStartAction,
  InitAction,
  RuntimeLoadAction,
  RefreshStartAction,
  RuntimeRegisterCompleteAction,
  RuntimeRegisterErrorAction,
  RuntimeRegisterStartAction,
  RuntimeUnregisterAction,
  RefreshCompleteAction,
  HostLoadAction,
  EnvLoadAction,
  EnvCreateCompleteAction,
} from './actions';
import {AppState, initialState} from './state';

type ActionType = string;
type Reducer = (action: any) => (prevState: AppState) => AppState;

const identityReducer = (action: Action) => (prevState: AppState) => prevState;

const reducers: {[key: ActionType]: Reducer} = {
  init: (action: InitAction) => (prevState: AppState) => initialState,
  'refresh-start': (action: RefreshStartAction) => prevState => ({
    ...prevState,
    runtimesLoadStatus: RuntimeViewStatus.refreshing,
    runtimes: [],
    hosts: [],
    envs: [],
  }),

  'refresh-complete': (action: RefreshCompleteAction) => prevState => ({
    ...prevState,
    runtimesLoadStatus: RuntimeViewStatus.done,
  }),

  'runtime-load': (action: RuntimeLoadAction) => prevState => {
    return {
      ...prevState,
      runtimes: [...prevState.runtimes, action.runtime],
    };
  },

  'host-load': (action: HostLoadAction) => prevState => {
    if (prevState.hosts.find(host => host.url === action.host.url)) {
      return prevState;
    }

    return {
      ...prevState,
      hosts: [...prevState.hosts, action.host],
    };
  },

  'env-load': (action: EnvLoadAction) => prevState => {
    const newEnv = action.env;
    if (
      prevState.envs.find(
        env =>
          env.hostUrl === newEnv.hostUrl && env.groupName === newEnv.groupName
      )
    ) {
      return prevState;
    }

    return {
      ...prevState,
      envs: [...prevState.envs, newEnv],
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
      hosts: prevState.hosts.filter(host => host.runtime !== action.alias),
      envs: prevState.envs.filter(env => env.runtimeAlias !== action.alias),
    };
  },

  'env-create-start': (action: EnvCreateStartAction) => prevState => {
    return {
      ...prevState,
      waits: {...prevState.waits, [action.wait.waitUrl]: action.wait},
    };
  },

  'env-create-complete': (action: EnvCreateCompleteAction) => prevState => {
    const newEnv = action.env;

    const alreadyHasNewEnv = !!prevState.envs.find(
      env =>
        env.hostUrl === newEnv.hostUrl && env.groupName === newEnv.groupName
    );

    return {
      ...prevState,
      envs: alreadyHasNewEnv ? prevState.envs : [...prevState.envs, newEnv],
      waits: Object.keys(prevState.waits)
        .filter(key => key !== action.waitUrl)
        .reduce((obj: {[key: string]: Wait}, key) => {
          obj[key] = prevState.waits[key];
          return obj;
        }, {}),
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

    const alreadyHasNewHost = !!prevState.hosts.find(
      host => host.url === newHost.url
    );

    return {
      ...prevState,
      hosts: alreadyHasNewHost
        ? prevState.hosts
        : [...prevState.hosts, newHost],
      waits: Object.keys(prevState.waits)
        .filter(key => key !== action.waitUrl)
        .reduce((obj: {[key: string]: Wait}, key) => {
          obj[key] = prevState.waits[key];
          return obj;
        }, {}),
    };
  },

  'host-delete-start': (action: HostDeleteStartAction) => prevState => {
    return {
      ...prevState,
      waits: {...prevState.waits, [action.wait.waitUrl]: action.wait},
    };
  },

  'host-delete-complete': (action: HostDeleteCompleteAction) => prevState => {
    const wait = prevState.waits[action.waitUrl];

    if (!isHostDeleteWait(wait)) {
      return prevState;
    }

    return {
      ...prevState,
      hosts: prevState.hosts.filter(host => host.url !== wait.metadata.hostUrl),
      envs: prevState.envs.filter(env => env.hostUrl !== wait.metadata.hostUrl),
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
