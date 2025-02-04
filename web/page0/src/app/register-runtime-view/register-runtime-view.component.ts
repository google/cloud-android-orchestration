import {Component, inject} from '@angular/core';
import {FormBuilder, Validators} from '@angular/forms';
import {
  ActivatedRoute,
  Event,
  NavigationEnd,
  Params,
  Router,
} from '@angular/router';
import {RuntimeService} from '../runtime.service';
import {MatSnackBar} from '@angular/material/snack-bar';
import {RuntimeViewStatus} from 'src/app/interface/runtime-interface';
import {of, Subject} from 'rxjs';
import {
  catchError,
  filter,
  map,
  mergeMap,
  shareReplay,
  switchMap,
  takeUntil,
  withLatestFrom,
} from 'rxjs/operators';
import {handleUrl} from '../utils';
import {Store} from 'src/app/store/store';
import {PLACEHOLDER_RUNTIME_SETTING} from '../settings';
import {runtimesLoadStatusSelector} from 'src/app/store/selectors';
import {FetchService} from '../fetch.service';

@Component({
  standalone: false,
  selector: 'app-register-runtime-view',
  templateUrl: './register-runtime-view.component.html',
  styleUrls: ['./register-runtime-view.component.scss'],
})
export class RegisterRuntimeViewComponent {
  private runtimeService = inject(RuntimeService);
  private formBuilder = inject(FormBuilder);
  private router = inject(Router);
  private snackBar = inject(MatSnackBar);
  private activatedRoute = inject(ActivatedRoute);
  private store = inject(Store);
  private fetchService = inject(FetchService);
  queryParams$ = this.router.events.pipe(filter((event: Event): event is NavigationEnd => event instanceof NavigationEnd), mergeMap(() => this.activatedRoute.queryParams), shareReplay(1));
  private ngUnsubscribe = new Subject<void>();
  previousUrl$ = this.queryParams$.pipe(map((params: Params) => (params['previousUrl'] ?? 'list-runtime') as string));
  runtimes$ = this.runtimeService.getRuntimes();
  status$ = this.store.select(runtimesLoadStatusSelector);

  constructor() {
    this.runtimeForm = this.formBuilder.group({
      url: [PLACEHOLDER_RUNTIME_SETTING.url, Validators.required],
      alias: [PLACEHOLDER_RUNTIME_SETTING.alias, Validators.required],
    });
    this.queryParams$.pipe(takeUntil(this.ngUnsubscribe)).subscribe();
  }

  runtimeForm;

  showProgressBar(status: RuntimeViewStatus | null) {
    return status === RuntimeViewStatus.registering;
  }

  ngOnInit() {
    this.runtimes$.pipe(takeUntil(this.ngUnsubscribe)).subscribe();
  }

  onSubmit() {
    const url = handleUrl(this.runtimeForm.value.url);
    const alias = this.runtimeForm.value.alias;

    if (!url || !alias) {
      return;
    }

    this.runtimeService
      .registerRuntime(alias, url)
      .pipe(
        withLatestFrom(this.previousUrl$),
        map(([runtime, previousUrl]) => {
          this.router.navigate([previousUrl]);
          this.snackBar.dismiss();
          return runtime;
        }),
        catchError(error => {
          this.snackBar.open(error.message, 'Dismiss');
          return of(undefined);
        }),
        switchMap(runtime => {
          if (!runtime) {
            return of();
          }

          return this.fetchService.loadHosts(runtime);
        })
      )
      .subscribe();
  }

  onCancel() {
    this.previousUrl$
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(previousUrl => {
        this.router.navigate([previousUrl]);
      });
  }

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }
}
