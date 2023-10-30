import {DeviceSetting} from 'src/app/interface/device-interface';
import {CVD, Group} from 'src/app/interface/host-orchestrator.dto';
import {RuntimeConfig} from './cloud-orchestrator.dto';
import {EnvConfig, Environment, EnvStatus} from './env-interface';
import {RuntimeInfo, RuntimeType} from './runtime-interface';

export function cvdToDevice(cvd: CVD): DeviceSetting {
  const {name, build_source} = cvd;

  const android_ci_build_source = build_source.android_ci_build_source;
  const main_build =
    android_ci_build_source ? android_ci_build_source.main_build : null;
  const branch = main_build ? main_build.branch : null;
  const build_id = main_build ? main_build.build_id : null;
  const target = main_build ? main_build.target : null;

  return {
    deviceId: name || 'unknown',
    branch_or_buildId: build_id || branch || 'unknown',
    target: target || 'unknown',
  };
};

export function groupToEnv(
  runtimeAlias: string,
  hostUrl: string,
  group: Group
): Environment {
  return {
    runtimeAlias,
    hostUrl,
    groupName: group.name || 'unknown',
    devices: group.cvds.map(cvdToDevice),
    status: EnvStatus.running,
  };
};

export function configToInfo(config: RuntimeConfig): RuntimeInfo {
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
