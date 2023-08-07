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
  takeUntil,
  tap,
  withLatestFrom,
} from 'rxjs';

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

  // TODO: get runtime from here directly instead of from createHost
  runtimeAlias$ = this.queryParams$.pipe(
    map((params) => (params['runtime'] ?? '') as string)
  );

  previousUrl$ = this.queryParams$.pipe(
    tap((previousUrl) => console.log('previousUrl: ', previousUrl)),
    map((params) => (params['previousUrl'] ?? 'list-runtime') as string)
  );

  isZoneActive = true;

  zones$ = new BehaviorSubject<string[]>(['us-central1-c', 'ap-northeast2-a']);

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
    const { machine_type, min_cpu_platform } = this.hostForm.value;
    if (!machine_type || !min_cpu_platform) {
      return;
    }

    this.hostService
      .createHost(
        {
          machine_type,
          min_cpu_platform,
        },
        'test'
      )
      .pipe(withLatestFrom(this.previousUrl$), takeUntil(this.ngUnsubscribe))
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
