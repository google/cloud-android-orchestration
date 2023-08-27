import {RuntimeViewStatus} from 'src/app/runtime-interface';
import {
  Action,
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
    return {
      ...prevState,
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
