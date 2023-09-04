import {Component} from '@angular/core';
import {FormBuilder} from '@angular/forms';
import {ActivatedRoute, NavigationEnd, Router} from '@angular/router';
import {HostService} from '../host.service';
import {MatSnackBar} from '@angular/material/snack-bar';
import {BehaviorSubject, Subject} from 'rxjs';
import {
  filter,
  map,
  mergeMap,
  shareReplay,
  switchMap,
  take,
  takeUntil,
  tap,
  withLatestFrom,
} from 'rxjs/operators';

import {Store} from 'src/app/store/store';
import {
  runtimeSelectorFactory,
  runtimesLoadStatusSelector,
} from '../store/selectors';
import {RuntimeViewStatus} from '../interface/runtime-interface';
import {defaultHostSetting, defaultZone} from '../settings';

@Component({
  selector: 'app-create-host-view',
  templateUrl: './create-host-view.component.html',
  styleUrls: ['./create-host-view.component.scss'],
})
export class CreateHostViewComponent {
  constructor(
    private hostService: HostService,
    private formBuilder: FormBuilder,
    private router: Router,
    private snackBar: MatSnackBar,
    private activatedRoute: ActivatedRoute,
    private store: Store
  ) {
    this.queryParams$.pipe(takeUntil(this.ngUnsubscribe)).subscribe();
  }

  private ngUnsubscribe = new Subject<void>();

  queryParams$ = this.router.events.pipe(
    filter((event): event is NavigationEnd => event instanceof NavigationEnd),
    mergeMap(() => this.activatedRoute.queryParams),
    shareReplay(1)
  );

  runtime$ = this.queryParams$.pipe(
    map(params => (params['runtime'] as string) ?? ''),
    switchMap(alias =>
      this.store.select(runtimesLoadStatusSelector).pipe(
        filter(status => status === RuntimeViewStatus.done),
        switchMap(() =>
          this.store.select(runtimeSelectorFactory({alias})).pipe(
            map(runtime => {
              if (!runtime) {
                throw new Error(`No runtime of alias ${alias}`);
              }
              return runtime;
            })
          )
        )
      )
    )
  );

  previousUrl$ = this.queryParams$.pipe(
    map(params => (params['previousUrl'] as string) ?? 'list-runtime')
  );

  zones$ = this.runtime$.pipe(
    map(runtime => runtime.zones),
    tap(zones => {
      if (zones?.includes(defaultZone)) {
        this.hostForm!.controls.zone.setValue(defaultZone);
      }
    })
  );

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  status$ = new BehaviorSubject<string>('done');

  hostForm = this.formBuilder.group({
    zone: ['ap-northeast2-a'],
    machine_type: [defaultHostSetting?.gcp?.machine_type],
    min_cpu_platform: [defaultHostSetting?.gcp?.min_cpu_platform],
  });

  // TODO: refactor with 'host status'
  showProgressBar(status: string | null) {
    return status === 'sending create request';
  }

  onSubmit() {
    const {machine_type, min_cpu_platform, zone} = this.hostForm.value;
    if (!machine_type || !min_cpu_platform || !zone) {
      return;
    }

    this.runtime$
      .pipe(
        take(1),
        switchMap(runtime => {
          this.status$.next('sending create request');
          return this.hostService.createHost(
            {
              gcp: {
                machine_type,
                min_cpu_platform,
              },
            },
            runtime,
            zone
          );
        }),
        withLatestFrom(this.previousUrl$),
        takeUntil(this.ngUnsubscribe)
      )
      .subscribe({
        next: ([_, previousUrl]) => {
          this.status$.next('done');
          this.router.navigate([previousUrl]);
          this.snackBar.dismiss();
        },
        error: error => {
          this.snackBar.open(error.message, 'dismiss');
        },
      });
  }

  onCancel() {
    this.previousUrl$
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(previousUrl => {
        this.router.navigate([previousUrl]);
      });
  }
}
