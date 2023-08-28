import {Injectable} from '@angular/core';
import {
  FormArray,
  FormBuilder,
  FormControl,
  FormGroup,
  Validators,
} from '@angular/forms';
import {Observable, Subject} from 'rxjs';
import {map, scan, startWith, tap} from 'rxjs/operators';
import {DeviceSetting} from './interface/device-interface';

interface DeviceInitAction {
  type: 'init';
}

interface DeviceClearAction {
  type: 'clear';
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
  | DeviceInitAction
  | DeviceClearAction;

type DeviceForm = FormArray<
  FormGroup<{
    deviceId: FormControl<string | null>;
    branch: FormControl<string | null>;
    target: FormControl<string | null>;
    buildId: FormControl<string | null>;
  }>
>;

@Injectable({
  providedIn: 'root',
})
export class DeviceFormService {
  private deviceFormAction$ = new Subject<DeviceFormAction>();

  addDevice() {
    this.deviceFormAction$.next({type: 'add'});
  }

  deleteDevice(targetIdx: number) {
    this.deviceFormAction$.next({type: 'delete', targetIdx});
  }

  duplicateDevice(targetIdx: number) {
    this.deviceFormAction$.next({type: 'duplicate', targetIdx});
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

  private toDeviceForm(setting: DeviceSetting) {
    return this.formBuilder.group({
      deviceId: [setting.deviceId, Validators.required],
      branch: [setting.branch, Validators.required],
      target: [setting.target, Validators.required],
      buildId: [setting.buildId, Validators.required],
    });
  }

  private toFormArray(deviceSettings: DeviceSetting[]): DeviceForm {
    return this.formBuilder.array(
      deviceSettings.map(setting => this.toDeviceForm(setting))
    );
  }

  private getInitDeviceForm(): DeviceForm {
    return this.toFormArray(this.getInitDeviceSettings());
  }

  private deviceSettingsForm$ = this.deviceFormAction$.pipe(
    tap((action: DeviceFormAction) => console.log('deviceForm ', action.type)),
    startWith({type: 'init'} as DeviceInitAction),
    scan((form: DeviceForm, action) => {
      if (action.type === 'init') {
        return form;
      }

      if (action.type === 'clear') {
        form.reset(this.getInitDeviceForm().value);
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
        const {branch, target, buildId} = form.at(action.targetIdx).value;

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
    }, this.getInitDeviceForm()),
    tap(form => console.log(form.value))
  );

  getDeviceSettingsForm() {
    return this.deviceSettingsForm$;
  }

  getValue(): Observable<DeviceSetting[]> {
    return this.deviceSettingsForm$.pipe(
      map((form: DeviceForm) => form.value),
      map(deviceSettings => {
        console.log(deviceSettings.length);
        return deviceSettings.map((setting, idx) => {
          const {deviceId, branch, target, buildId} = setting;

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

  clearForm() {
    this.deviceFormAction$.next({type: 'clear'});
  }

  constructor(private formBuilder: FormBuilder) {}
}
