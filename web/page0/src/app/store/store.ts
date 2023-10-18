import {Injectable} from '@angular/core';
import {Observable, Subject} from 'rxjs';
import {map, scan, shareReplay, startWith} from 'rxjs/operators';
import {Action, InitAction} from './actions';
import {match} from './reducers';
import {AppState, INITIAL_STATE} from './state';

@Injectable({
  providedIn: 'root',
})
export class Store {
  constructor() {}

  action$ = new Subject<Action>();

  // should be updated by all reducers
  private state$ = this.action$.pipe(
    startWith({type: 'init'} as InitAction),
    map(action => {
      return match(action);
    }),
    scan((prevState, handler) => handler(prevState), INITIAL_STATE),
    shareReplay(1)
  );

  select<T>(selector: (state: AppState) => T): Observable<T> {
    return this.state$.pipe(map(state => selector(state)));
  }

  dispatch(action: Action) {
    this.action$.next(action);
  }
}
