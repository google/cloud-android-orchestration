import {Injectable} from '@angular/core';
import {Subject} from 'rxjs';
import {map, scan, startWith, tap} from 'rxjs/operators';
import {match} from './reducers';
import {AppState, initialState} from './state';

@Injectable({
  providedIn: 'root',
})
export class Store {
  constructor() {}

  private actionSubject = new Subject<Action>();

  // should be updated by all reducers
  private state$ = this.actionSubject.pipe(
    tap(action => console.log(action.type)),
    startWith({type: 'init'} as InitAction),
    map(action => {
      return match(action);
    }),
    scan((prevState, handler) => handler(prevState), initialState)
  );

  select<T>(selector: (state: AppState) => T) {
    return this.state$.pipe(map(state => selector(state)));
  }

  dispatch<T>(action: Action) {
    this.actionSubject.next(action);
  }
}
