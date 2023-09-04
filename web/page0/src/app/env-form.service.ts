import {Injectable} from '@angular/core';
import {FormArray, FormBuilder, FormControl, FormGroup} from '@angular/forms';
import {of} from 'rxjs';
import {startWith, switchMap, tap} from 'rxjs/operators';
import {Store} from 'src/app/store/store';
import {defaultEnvConfig, defaultZone} from './settings';
import {parseEnvConfig} from './interface/utils';
import {DeviceSetting} from './interface/device-interface';
import jsonutils from './json.utils';
import {adjustArrayLength} from './utils';

type DeviceForm = FormArray<
  FormGroup<{
    deviceId: FormControl<string | null>;
    branch_or_buildId: FormControl<string | null>;
    target: FormControl<string | null>;
  }>
>;

type EnvForm = FormGroup<{
  canonicalConfig: FormControl<string | null>;
  groupName: FormControl<string | null>;
  runtime: FormControl<string | null>;
  zone: FormControl<string | null>;
  host: FormControl<string | null>;
  devices: DeviceForm;
}>;

@Injectable({
  providedIn: 'root',
})
export class EnvFormService {
  constructor(
    private formBuilder: FormBuilder,
    private store: Store
  ) {
    this.initialize();
  }

  private envForm: EnvForm | undefined;

  private initialize() {
    this.envForm = this.getInitEnvForm();
    this.syncConfig();
  }

  private syncConfig() {
    const options = {emitEvent: false};
    const controls = this.envForm!.controls;

    controls.canonicalConfig.valueChanges
      .pipe(
        startWith(controls.canonicalConfig.value),
        tap(config => {
          try {
            const {groupName, devices} = parseEnvConfig(config);

            controls.groupName.setValue(groupName, options);

            const deviceControls = controls.devices;
            const instanceNum = devices.length;
            const deviceControlNum = deviceControls.length;

            for (let cnt = deviceControlNum; cnt < instanceNum; cnt++) {
              deviceControls.push(
                this.toDeviceForm({
                  deviceId: 'cvd',
                  branch_or_buildId: '',
                  target: '',
                }),
                options
              );
            }

            for (let cnt = deviceControlNum - 1; cnt >= instanceNum; cnt--) {
              deviceControls.removeAt(cnt, options);
            }

            devices.forEach((device, idx) => {
              deviceControls.at(idx).setValue(device, options);
            });

            return;
          } catch (error) {
            controls.canonicalConfig.setErrors({invalid: true, message: error});
            return;
          }
        })
      )
      .subscribe();

    controls.groupName.valueChanges
      .pipe(
        startWith(controls.groupName.value),
        tap(groupName => {
          const config = controls.canonicalConfig;
          try {
            const object = jsonutils.parse(config.value);
            jsonutils.setValue(object, ['common', 'group_name'], groupName);
            config.setValue(jsonutils.stringify(object), options);
          } catch (error) {
            return;
          }
        })
      )
      .subscribe();

    controls.devices.valueChanges
      .pipe(
        tap(devices => {
          const config = controls.canonicalConfig;
          try {
            const object = jsonutils.parse(config.value);

            jsonutils.setValue(object, ['instances'], [], {ifexists: 'skip'});
            adjustArrayLength<object>(object['instances'], devices.length, {});

            for (let i = 0; i < devices.length; i++) {
              const device = devices[i];
              const deviceId = device.deviceId || '';
              const target = device.target || '';
              const branch_or_buildId = device.branch_or_buildId || '';

              jsonutils.setValue(
                object,
                ['instances', `${i}`, 'name'],
                deviceId
              );

              jsonutils.setValue(
                object,
                ['instances', `${i}`, 'disk', 'default_build'],
                `@ab/${branch_or_buildId}/${target}`
              );
            }

            config.setValue(jsonutils.stringify(object), options);
          } catch (error) {
            return;
          }
        })
      )
      .subscribe();
  }

  private getInitEnvForm(): EnvForm {
    const initCanonicalConfig = jsonutils.stringify(defaultEnvConfig);

    return this.formBuilder.group({
      canonicalConfig: [initCanonicalConfig],
      groupName: [''],
      runtime: [''],
      zone: [''],
      host: [''],
      devices: this.getInitDeviceForm([]),
    });
  }

  private getInitDeviceForm(deviceSettings: DeviceSetting[]): DeviceForm {
    return this.formBuilder.array(
      deviceSettings.map(setting => this.toDeviceForm(setting))
    );
  }

  private toDeviceForm(setting: DeviceSetting) {
    return this.formBuilder.group({
      deviceId: [setting.deviceId],
      branch_or_buildId: [setting.branch_or_buildId],
      target: [setting.target],
    });
  }

  getEnvForm() {
    return this.envForm!;
  }

  getZones$() {
    return (
      this.envForm!.controls.runtime.valueChanges.pipe(
        startWith(this.envForm!.value.runtime),
        switchMap(selectedRuntimeAlias => {
          return this.store.select(state => {
            const selectedRuntime = state.runtimes.find(
              runtime => runtime.alias === selectedRuntimeAlias
            );
            return selectedRuntime?.zones || [];
          });
        }),
        tap(zones => {
          if (zones.includes(defaultZone)) {
            this.envForm!.controls.zone.setValue(defaultZone);
          }
        })
      ) || of([])
    );
  }

  getHosts$() {
    return (
      this.envForm!.valueChanges.pipe(
        startWith({
          runtime: this.envForm!.value.runtime,
          zone: this.envForm!.value.zone,
        }),
        switchMap(({runtime, zone}) => {
          return this.store.select(state => {
            return state.hosts
              .filter(host => host.runtime === runtime && host.zone === zone)
              .map(host => host.name);
          });
        })
      ) || of([])
    );
  }

  clearForm() {
    this.envForm!.reset();
    this.initialize();
  }

  addDevice() {
    this.envForm!.controls.devices.push(
      this.formBuilder.group({
        deviceId: [`cvd-${this.envForm!.controls.devices.length + 1}`],
        branch_or_buildId: [''],
        target: [''],
      })
    );
  }

  deleteDevice(targetIdx: number) {
    this.envForm!.controls.devices.removeAt(targetIdx);
  }

  duplicateDevice(targetIdx: number) {
    const {branch_or_buildId, target} =
      this.envForm!.controls.devices.at(targetIdx).value;

    this.envForm!.controls.devices.push(
      this.toDeviceForm({
        deviceId: `cvd-${this.envForm!.controls.devices.length + 1}`,
        branch_or_buildId: branch_or_buildId ?? '',
        target: target ?? '',
      })
    );
  }
}
