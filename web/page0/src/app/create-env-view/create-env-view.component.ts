import { Component } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router } from '@angular/router';
import { map, Subject, switchMap, takeUntil, tap, withLatestFrom } from 'rxjs';
import { DeviceFormService } from '../device-form.service';
import { EnvFormService } from '../env-form.service';
import { EnvService } from '../env.service';

@Component({
  selector: 'app-create-env-view',
  templateUrl: './create-env-view.component.html',
  styleUrls: ['./create-env-view.component.scss'],
})
export class CreateEnvViewComponent {
  envForm$ = this.envFormService.getEnvForm();
  runtimes$ = this.envFormService.runtimes$;
  zones$ = this.envFormService.zones$;
  hosts$ = this.envFormService.hosts$;
  deviceSettingsForm$ = this.deviceFormService.getDeviceSettingsForm();

  constructor(
    private router: Router,
    private snackBar: MatSnackBar,
    private deviceFormService: DeviceFormService,
    private envService: EnvService,
    private envFormService: EnvFormService
  ) {}

  private ngUnsubscribe = new Subject<void>();

  ngOnInit() {
    this.zones$.subscribe();
  }

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  onClickAddDevice() {
    this.deviceFormService.addDevice();
  }

  onClickRegisterRuntime() {
    this.router.navigate(['/register-runtime'], {
      queryParams: {
        previousUrl: 'create-env',
      },
    });
  }

  onClickCreateHost() {
    this.envFormService
      .getSelectedRuntime()
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe((runtime) => {
        this.router.navigate(['/create-host'], {
          queryParams: {
            previousUrl: 'create-env',
            runtime,
          },
        });
      });
  }

  onSubmit() {
    this.envFormService
      .getValue()
      .pipe(
        switchMap(({ groupName, hostUrl }) =>
          this.deviceFormService.getValue().pipe(
            switchMap((devices) =>
              this.envService.createEnv(hostUrl, {
                groupName,
                devices,
              })
            )
          )
        ),
        takeUntil(this.ngUnsubscribe)
      )
      .subscribe({
        next: () => {
          this.snackBar.dismiss();
          this.envFormService.clearForm();
          this.deviceFormService.clearForm();
          this.router.navigate(['/']);
        },
        error: (error) => {
          this.snackBar.open(error.message);
        },
      });
  }

  onCancel() {
    this.router.navigate(['/']);
    this.envFormService.clearForm();
    this.deviceFormService.clearForm();
  }
}
