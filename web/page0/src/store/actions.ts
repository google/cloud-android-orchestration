type Action = InitAction | IncrementAction | DecrementAction;

interface InitAction {
  type: 'init';
  value: number;
}

interface IncrementAction {
  type: 'increment';
}

interface DecrementAction {
  type: 'decrement';
}
