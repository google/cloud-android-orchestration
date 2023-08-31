import {Component} from '@angular/core';
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
import {Subject} from 'rxjs';
import {
  filter,
  map,
  mergeMap,
  shareReplay,
  takeUntil,
  withLatestFrom,
} from 'rxjs/operators';
import {handleUrl} from '../utils';
import {Store} from 'src/app/store/store';
import {placeholderRuntimeSetting} from '../settings';
import {runtimesLoadStatusSelector} from 'src/app/store/selectors';

@Component({
  selector: 'app-register-runtime-view',
  templateUrl: './register-runtime-view.component.html',
  styleUrls: ['./register-runtime-view.component.scss'],
})
export class RegisterRuntimeViewComponent {
  constructor(
    private runtimeService: RuntimeService,
    private formBuilder: FormBuilder,
    private router: Router,
    private snackBar: MatSnackBar,
    private activatedRoute: ActivatedRoute,
    private store: Store
  ) {
    this.queryParams$.pipe(takeUntil(this.ngUnsubscribe)).subscribe();
  }

  queryParams$ = this.router.events.pipe(
    filter(
      (event: Event): event is NavigationEnd => event instanceof NavigationEnd
    ),
    mergeMap(() => this.activatedRoute.queryParams),
    shareReplay(1)
  );

  private ngUnsubscribe = new Subject<void>();

  previousUrl$ = this.queryParams$.pipe(
    map((params: Params) => (params['previousUrl'] ?? 'list-runtime') as string)
  );

  runtimes$ = this.runtimeService.getRuntimes();
  status$ = this.store.select(runtimesLoadStatusSelector);

  runtimeForm = this.formBuilder.group({
    url: [placeholderRuntimeSetting.url, Validators.required],
    alias: [placeholderRuntimeSetting.alias, Validators.required],
  });

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
      .pipe(withLatestFrom(this.previousUrl$), takeUntil(this.ngUnsubscribe))
      .subscribe({
        next: ([_, previousUrl]) => {
          this.router.navigate([previousUrl]);
          this.snackBar.dismiss();
        },
        error: error => {
          this.snackBar.open(error.message);
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

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }
}
