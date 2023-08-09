import { Component } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router } from '@angular/router';
import { combineLatestWith, map, shareReplay, switchMap, tap } from 'rxjs';
import { DeviceFormService } from '../device-form.service';
import { EnvService } from '../env.service';
import { HostService } from '../host.service';
import { RuntimeService } from '../runtime.service';

@Component({
  selector: 'app-create-env-view',
  templateUrl: './create-env-view.component.html',
  styleUrls: ['./create-env-view.component.scss'],
})
export class CreateEnvViewComponent {
  envForm = this.formBuilder.group({
    groupName: ['', Validators.required],
    runtime: ['', Validators.required],
    zone: ['', Validators.required],
    host: ['', Validators.required],
  });

  runtimes$ = this.runtimeService.getRuntimes();

  selectedRuntime$ = this.envForm.controls['runtime'].valueChanges.pipe(
    map((alias) => alias ?? ''),
    switchMap((alias: string) => this.runtimeService.getRuntimeByAlias(alias))
  );

  zones$ = this.selectedRuntime$.pipe(map((runtime) => runtime?.zones || []));

  selectedZone$ = this.envForm.controls['zone'].valueChanges.pipe(
    map((zone) => zone ?? '')
  );

  hosts$ = this.selectedZone$.pipe(
    combineLatestWith(this.selectedRuntime$),
    switchMap(([zone, runtime]) =>
      this.hostService.getHostsByZone(runtime.alias, zone)
    ),
    shareReplay(1)
  );

  deviceSettingsForm$ = this.deviceFormService.getDeviceSettingsForm();

  // TODO: restore current progress from local storage
  constructor(
    private formBuilder: FormBuilder,
    private router: Router,
    private snackBar: MatSnackBar,
    private runtimeService: RuntimeService,
    private hostService: HostService,
    private deviceFormService: DeviceFormService,
    private envService: EnvService
  ) {}

  ngOnInit() {}

  onClickAddDevice() {
    this.deviceFormService.addDevice();
  }

  onClickRegisterRuntime() {
    // TODO: save current progress in local storage
    this.router.navigate(['/register-runtime'], {
      queryParams: {
        previousUrl: 'create-env',
      },
    });
  }

  onClickCreateHost() {
    // TODO: save current progress in local storage
    this.router.navigate(['/create-host'], {
      queryParams: {
        previousUrl: 'create-env',
      },
    });
  }

  onSubmit() {
    console.log(this.envForm.value);

    const { host: hostName, groupName } = this.envForm.value;
    if (!hostName || !groupName) {
      return;
    }

    this.hosts$
      .pipe(
        map((hosts) => hosts.find((host) => host.name === hostName)),
        switchMap((host) => {
          if (!host) {
            throw new Error(`No host of name ${hostName}`);
          }

          return this.deviceSettingsForm$.pipe(
            map((form) => form.value),
            switchMap((deviceSettings) => {
              const devices = deviceSettings.map((setting, idx) => {
                const { deviceId, branch, target, buildId } = setting;

                if (!deviceId || !branch || !target || !buildId) {
                  throw new Error(`Device # ${idx + 1} has empty field`);
                }

                return {
                  deviceId,
                  branch,
                  target,
                  buildId,
                };
              });

              return this.envService.createEnv(host.url, {
                groupName,
                devices,
              });
            })
          );
        })
      )
      .subscribe({
        next: () => {
          this.snackBar.dismiss();
          this.router.navigate(['/']);
        },
        error: (error) => {
          this.snackBar.open(error.message);
        },
      });
  }

  onCancel() {
    this.router.navigate(['/']);
  }
}
