import {Injectable} from '@angular/core';
import {FormBuilder, Validators} from '@angular/forms';
import {
  catchError,
  combineLatestWith,
  map,
  of,
  scan,
  shareReplay,
  startWith,
  Subject,
  switchMap,
  tap,
} from 'rxjs';
import {Store} from 'src/store/store';
import {
  hostListSelectorFactory,
  hostSelectorFactory,
  runtimeListSelector,
} from 'src/store/selectors';

interface EnvFormInitAction {
  type: 'init';
}

interface EnvFormClearAction {
  type: 'clear';
}

type EnvFormAction = EnvFormInitAction | EnvFormClearAction;

@Injectable({
  providedIn: 'root',
})
export class EnvFormService {
  constructor(
    private formBuilder: FormBuilder,
    private store: Store
  ) {}

  private envFormAction$ = new Subject<EnvFormAction>();

  private envForm$ = this.envFormAction$.pipe(
    startWith({type: 'init'} as EnvFormInitAction),
    scan((form, action) => {
      if (action.type === 'init') {
        return form;
      }

      if (action.type === 'clear') {
        form.reset();
        return form;
      }

      return form;
    }, this.getInitEnvForm())
  );

  getInitEnvForm() {
    return this.formBuilder.group({
      groupName: ['', Validators.required],
      runtime: ['', Validators.required],
      zone: ['', Validators.required],
      host: ['', Validators.required],
    });
  }

  getEnvForm() {
    return this.envForm$;
  }

  runtimes$ = this.store.select(runtimeListSelector);

  private selectedRuntime$ = this.envForm$.pipe(
    switchMap(form => {
      return form.controls.runtime.valueChanges.pipe(
        map(alias => alias ?? ''),
        tap(alias => console.log(`selected runtime: ${alias}`)),
        tap(() => {
          form.controls.zone.setValue('');
        }),
        switchMap((alias: string) =>
          this.store
            .select(state =>
              state.runtimes.find(runtime => runtime.alias === alias)
            )
            .pipe(
              map(runtime => {
                if (!runtime) {
                  throw new Error(`No runtime of alias ${alias}`);
                }
                return runtime;
              })
            )
        ),
        catchError(error => {
          console.error(error);
          return of();
        })
      );
    }),
    shareReplay(1)
  );

  zones$ = this.selectedRuntime$.pipe(
    map(runtime => runtime?.zones || []),
    tap(zones => console.log('zones: ', zones.length))
  );

  private selectedZone$ = this.envForm$.pipe(
    switchMap(form =>
      form.controls.zone.valueChanges.pipe(
        map(zone => zone ?? ''),
        tap(zone => console.log('selected zone: ', zone)),
        tap(() => {
          form.controls.host.setValue('');
        })
      )
    ),
    shareReplay(1)
  );

  hosts$ = this.selectedZone$.pipe(
    combineLatestWith(this.selectedRuntime$),
    switchMap(([zone, runtime]) => {
      if (!runtime) {
        return of([]);
      }
      return this.store.select(
        hostListSelectorFactory({runtimeAlias: runtime.alias, zone})
      );
    }),
    tap(hosts => console.log('hosts: ', hosts.length))
  );

  getSelectedRuntime() {
    return this.envForm$.pipe(map(form => form.value.runtime));
  }

  getValue() {
    return this.envForm$.pipe(
      switchMap(form => {
        const {runtime, zone, groupName, host} = form.value;
        if (!runtime || !zone || !groupName || !host) {
          console.error(form.value);
          throw new Error(
            'Group name, runtime, zone, host should be specified'
          );
        }

        return this.store
          .select(
            hostSelectorFactory({runtimeAlias: runtime, zone, name: host})
          )
          .pipe(
            map(host => {
              if (!host) {
                throw new Error('Invalid host');
              }
              return {
                groupName,
                hostUrl: host.url,
                runtime: host.runtime,
              };
            })
          );
      })
    );
  }

  clearForm() {
    this.envFormAction$.next({type: 'clear'});
  }
}
