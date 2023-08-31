import {Injectable} from '@angular/core';
import {Observable, Subject} from 'rxjs';
import {map, scan, shareReplay, startWith, tap} from 'rxjs/operators';
import {Action, InitAction} from './actions';
import {match} from './reducers';
import {AppState, initialState} from './state';

@Injectable({
  providedIn: 'root',
})
export class Store {
  constructor() {}

  action$ = new Subject<Action>();

  // should be updated by all reducers
  private state$ = this.action$.pipe(
    tap(action => console.log('action ', action.type)),
    startWith({type: 'init'} as InitAction),
    map(action => {
      return match(action);
    }),
    scan((prevState, handler) => handler(prevState), initialState),
    shareReplay(1)
  );

  select<T>(selector: (state: AppState) => T): Observable<T> {
    return this.state$.pipe(map(state => selector(state)));
  }

  dispatch(action: Action) {
    this.action$.next(action);
  }
}
