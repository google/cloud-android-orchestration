import { Injectable } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { map, scan, startWith, Subject, tap } from 'rxjs';
import { DeviceSetting } from './device-interface';

interface DeviceInitAction {
  type: 'init';
}

interface DeviceAddAction {
  type: 'add';
}

interface DeviceDuplicateAction {
  type: 'duplicate';
  targetIdx: number;
}

interface DeviceDeleteAction {
  type: 'delete';
  targetIdx: number;
}

type DeviceFormAction =
  | DeviceAddAction
  | DeviceDuplicateAction
  | DeviceDeleteAction
  | DeviceInitAction;

@Injectable({
  providedIn: 'root',
})
export class DeviceFormService {
  private deviceFormAction$ = new Subject<DeviceFormAction>();

  addDevice() {
    this.deviceFormAction$.next({ type: 'add' });
  }

  deleteDevice(targetIdx: number) {
    this.deviceFormAction$.next({ type: 'delete', targetIdx });
  }

  duplicateDevice(targetIdx: number) {
    this.deviceFormAction$.next({ type: 'duplicate', targetIdx });
  }

  private getInitDeviceSettings() {
    return [
      {
        deviceId: 'cvd-1',
        branch: 'default',
        target: 'default',
        buildId: 'default',
      },
    ];
  }

  toDeviceForm(setting: DeviceSetting) {
    return this.formBuilder.group({
      deviceId: [setting.deviceId, Validators.required],
      branch: [setting.branch, Validators.required],
      target: [setting.target, Validators.required],
      buildId: [setting.buildId, Validators.required],
    });
  }

  toFormArray(deviceSettings: DeviceSetting[]) {
    return this.formBuilder.array(
      deviceSettings.map((setting) => this.toDeviceForm(setting))
    );
  }

  private deviceSettingsForm$ = this.deviceFormAction$.pipe(
    startWith({ type: 'init' } as DeviceInitAction),
    scan((form, action) => {
      if (action.type === 'init') {
        return form;
      }

      if (action.type === 'add') {
        form.push(
          this.toDeviceForm({
            deviceId: `cvd-${form.length + 1}`,
            branch: '',
            target: '',
            buildId: '',
          })
        );
        return form;
      }

      if (action.type === 'delete') {
        form.removeAt(action.targetIdx);
        return form;
      }

      if (action.type === 'duplicate') {
        const { branch, target, buildId } = form.at(action.targetIdx).value;

        form.push(
          this.toDeviceForm({
            deviceId: `cvd-${form.length + 1}`,
            branch: branch ?? '',
            target: target ?? '',
            buildId: buildId ?? '',
          })
        );

        return form;
      }

      return form;
    }, this.toFormArray(this.getInitDeviceSettings())),
    tap((form) => console.log(form.value))
  );

  getDeviceSettingsForm() {
    return this.deviceSettingsForm$;
  }

  getValue() {
    return this.deviceSettingsForm$.pipe(
      map((form) => form.value),
      tap((v) => console.log(v)),
      map((deviceSettings) => {
        console.log(deviceSettings.length);
        return deviceSettings.map((setting, idx) => {
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
      })
    );
  }

  // TODO: clear form

  constructor(private formBuilder: FormBuilder) {}
}
