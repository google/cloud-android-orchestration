import {AppState, initialState} from './state';

type ActionType = string;
type Reducer = (action: any) => (prevState: AppState) => AppState;

const identityReducer = (action: Action) => (prevState: AppState) => prevState;

const reducers: {[key: ActionType]: Reducer} = {
  init: (action: InitAction) => (prevState: AppState) => initialState,
  tmp: (action: InitAction) => (prevState: AppState) => ({boo: 3}),
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
