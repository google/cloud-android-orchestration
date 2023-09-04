import {Component} from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';
import {Router} from '@angular/router';
import {BehaviorSubject} from 'rxjs';
import {map, tap} from 'rxjs/operators';
import {EnvFormService} from '../env-form.service';
import {EnvService} from '../env.service';
import {validRuntimeListSelector} from '../store/selectors';
import {Store} from '../store/store';
@Component({
  selector: 'app-create-env-view',
  templateUrl: './create-env-view.component.html',
  styleUrls: ['./create-env-view.component.scss'],
})
export class CreateEnvViewComponent {
  envForm = this.envFormService.getEnvForm();

  constructor(
    private router: Router,
    private snackBar: MatSnackBar,
    private envService: EnvService,
    private envFormService: EnvFormService,
    private store: Store
  ) {}

  runtimes$ = this.store
    .select(validRuntimeListSelector)
    .pipe(map(runtimes => runtimes.map(runtime => runtime.alias)));

  zones$ = this.envFormService.getZones$();

  hosts$ = this.envFormService.getHosts$();

  status$ = new BehaviorSubject<string>('done');

  showProgressBar(status: string | null) {
    return status === 'sending create request';
  }

  onClickAddDevice() {
    this.envFormService.addDevice();
  }

  onClickRegisterRuntime() {
    this.router.navigate(['/register-runtime'], {
      queryParams: {
        previousUrl: 'create-env',
      },
    });
  }

  onClickCreateHost() {
    this.router.navigate(['/create-host'], {
      queryParams: {
        previousUrl: 'create-env',
        runtime: this.envForm.value.runtime,
      },
    });
  }

  onSubmit() {
    const {runtime, zone, host, canonicalConfig} = this.envForm.value;

    this.status$.next('sending create request');
    this.envService
      .createEnv(runtime, zone, host, canonicalConfig)
      .pipe(tap(() => this.status$.next('done')))
      .subscribe({
        next: () => {
          this.snackBar.dismiss();
          this.router.navigate(['/']);
          this.envFormService.clearForm();
        },
        error: error => {
          this.snackBar.open(error.message);
        },
      });
  }

  onCancel() {
    this.router.navigate(['/']);
    this.envFormService.clearForm();
  }
}
