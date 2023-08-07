import { Component } from '@angular/core';
import { FormBuilder } from '@angular/forms';
import { ActivatedRoute, NavigationEnd, Router } from '@angular/router';
import { HostService } from '../host.service';
import { MatSnackBar } from '@angular/material/snack-bar';
import {
  BehaviorSubject,
  filter,
  map,
  mergeMap,
  shareReplay,
  Subject,
  switchMap,
  takeUntil,
  tap,
  withLatestFrom,
} from 'rxjs';
import { RuntimeService } from '../runtime.service';

@Component({
  selector: 'app-create-host-view',
  templateUrl: './create-host-view.component.html',
  styleUrls: ['./create-host-view.component.scss'],
})
export class CreateHostViewComponent {
  constructor(
    private runtimeService: RuntimeService,
    private hostService: HostService,
    private formBuilder: FormBuilder,
    private router: Router,
    private snackBar: MatSnackBar,
    private activatedRoute: ActivatedRoute
  ) {
    this.queryParams$
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe((params) => {
        console.log(params);
      });
  }

  private ngUnsubscribe = new Subject<void>();

  queryParams$ = this.router.events.pipe(
    filter((event): event is NavigationEnd => event instanceof NavigationEnd),
    mergeMap(() => this.activatedRoute.queryParams),
    shareReplay(1)
  );

  runtime$ = this.queryParams$.pipe(
    map((params) => (params['runtime'] ?? '') as string),
    switchMap((alias) => this.runtimeService.getRuntimeByAlias(alias))
  );

  previousUrl$ = this.queryParams$.pipe(
    tap((previousUrl) => console.log('previousUrl: ', previousUrl)),
    map((params) => (params['previousUrl'] ?? 'list-runtime') as string)
  );

  zones$ = this.runtime$.pipe(map((runtime) => runtime.zones));

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  status$ = new BehaviorSubject<string>('done');

  hostForm = this.formBuilder.group({
    zone: ['ap-northeast2-a'],
    machine_type: ['n1-standard-4'],
    min_cpu_platform: ['Intel Skylake'],
  });

  // TODO: refactor with 'host status'
  showProgressBar(status: string | null) {
    return status === 'registering';
  }

  onSubmit() {
    const { machine_type, min_cpu_platform, zone } = this.hostForm.value;
    if (!machine_type || !min_cpu_platform || !zone) {
      return;
    }

    this.runtime$
      .pipe(
        switchMap((runtime) =>
          this.hostService.createHost(
            {
              gcp: {
                machine_type,
                min_cpu_platform,
              },
            },
            runtime,
            zone
          )
        ),
        withLatestFrom(this.previousUrl$),
        takeUntil(this.ngUnsubscribe)
      )
      .subscribe({
        next: ([_, previousUrl]) => {
          this.router.navigate([previousUrl]);
          this.snackBar.dismiss();
        },
        error: (error) => {
          this.snackBar.open(error.message);
        },
      });
  }

  onCancel() {
    this.previousUrl$
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe((previousUrl) => {
        this.router.navigate([previousUrl]);
      });
  }
}
