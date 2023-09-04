import {DeviceSetting} from 'src/app/interface/device-interface';
import {CVD} from 'src/app/interface/host-orchestrator.dto';
import {RuntimeConfig} from './cloud-orchestrator.dto';
import {EnvConfig} from './env-interface';
import {RuntimeInfo, RuntimeType} from './runtime-interface';

export const cvdToDevice = (cvd: CVD): DeviceSetting => {
  const {name, build_source} = cvd;
  const {android_ci_build_source} = build_source;
  const {main_build} = android_ci_build_source;
  const {branch, build_id, target} = main_build;

  return {
    deviceId: name,
    branch_or_buildId: build_id || branch,
    target,
  };
};

export const configToInfo = (config: RuntimeConfig): RuntimeInfo => {
  const {instance_manager_type} = config;
  if (instance_manager_type === 'GCP') {
    return {
      type: RuntimeType.cloud,
    };
  }

  return {
    type: RuntimeType.local,
  };
};

export function envConfigToEnv(config: EnvConfig): {
  groupName: string;
  devices: DeviceSetting[];
} {
  return {
    groupName: config?.common?.group_name || '',
    devices:
      config?.instances?.map(instance => {
        const [_, branch_or_buildId, target] =
          instance.disk.default_build.split('/');

        return {
          deviceId: instance?.name || '',
          branch_or_buildId,
          target,
        };
      }) || [],
  };
}

export function deviceToInstanceConfig(device: DeviceSetting) {
  return {
    name: device.deviceId,
    disk: {
      default_build: `@ab/${device.branch_or_buildId}/${device.target}`,
    },
  };
}

export function parseEnvConfig(canonicalConfig: string | null | undefined) {
  if (!canonicalConfig) {
    throw new Error('Cannot parse empty string');
  }

  const config = JSON.parse(canonicalConfig) as EnvConfig;
  return envConfigToEnv(config);
}
