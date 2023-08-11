import { Component } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { ActivatedRoute, NavigationEnd, Router } from '@angular/router';
import { RuntimeService } from '../runtime.service';
import { MatSnackBar } from '@angular/material/snack-bar';
import { RuntimeViewStatus } from '../runtime-interface';
import {
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
    private activatedRoute: ActivatedRoute
  ) {
    this.queryParams$.pipe(takeUntil(this.ngUnsubscribe)).subscribe();
  }

  queryParams$ = this.router.events.pipe(
    filter((event): event is NavigationEnd => event instanceof NavigationEnd),
    mergeMap(() => this.activatedRoute.queryParams),
    shareReplay(1)
  );

  private ngUnsubscribe = new Subject<void>();

  previousUrl$ = this.queryParams$.pipe(
    tap((previousUrl) => console.log('previousUrl: ', previousUrl)),
    map((params) => (params['previousUrl'] ?? 'list-runtime') as string)
  );

  runtimes$ = this.runtimeService.getRuntimes();
  status$ = this.runtimeService.getStatus();

  runtimeForm = this.formBuilder.group({
    url: ['http://localhost:8071/api', Validators.required],
    alias: ['test', Validators.required],
  });

  showProgressBar(status: RuntimeViewStatus | null) {
    return status === RuntimeViewStatus.registering;
  }

  onSubmit() {
    const url = this.runtimeForm.value.url;
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

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }
}
