import { Component } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router } from '@angular/router';
import { combineLatestWith, map, switchMap } from 'rxjs';
import { DeviceFormService } from '../device-form.service';
import { HostService } from '../host.service';
import { RuntimeService } from '../runtime.service';

@Component({
  selector: 'app-create-env-view',
  templateUrl: './create-env-view.component.html',
  styleUrls: ['./create-env-view.component.scss'],
})
export class CreateEnvViewComponent {
  envForm = this.formBuilder.group({
    groupId: ['', Validators.required],
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
    )
  );

  deviceSettingsForm$ = this.deviceFormService.getDeviceSettingsForm();

  // TODO: restore current progress from local storage
  constructor(
    private formBuilder: FormBuilder,
    private router: Router,
    private snackBar: MatSnackBar,
    private runtimeService: RuntimeService,
    private hostService: HostService,
    private deviceFormService: DeviceFormService
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
    // TODO: validation
    // TODO: Call POST /cvds using options
    console.log(this.envForm.value);
    this.router.navigate(['/']);
  }

  onCancel() {
    this.router.navigate(['/']);
  }
}
