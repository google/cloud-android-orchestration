export enum ResultType {
  waitStarted = 'wait-started',
  done = 'done',
}

export interface WaitStartedResult {
  type: ResultType.waitStarted;
  waitUrl: string;
}

export interface DoneResult<T> {
  type: ResultType.done;
  waitUrl: string;
  data: T;
}

export type Result<T> = DoneResult<T> | WaitStartedResult;
