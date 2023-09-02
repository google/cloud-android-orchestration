import {Component} from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';
import {Router} from '@angular/router';
import {map} from 'rxjs/operators';
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
    const {runtime, host, canonicalConfig} = this.envForm.value;

    this.envService.createEnv(runtime, host, canonicalConfig).subscribe({
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
