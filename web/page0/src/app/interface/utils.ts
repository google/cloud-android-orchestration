import {DeviceSetting} from 'src/app/interface/device-interface';
import {CVD} from 'src/app/interface/host-orchestrator.dto';
import {RuntimeConfig} from './cloud-orchestrator.dto';
import {RuntimeInfo, RuntimeType} from './runtime-interface';

export const cvdToDevice = (cvd: CVD): DeviceSetting => {
  const {name, build_source} = cvd;
  const {android_ci_build_source} = build_source;
  const {main_build} = android_ci_build_source;
  const {branch, build_id, target} = main_build;

  return {
    deviceId: name,
    branch,
    buildId: build_id,
    target,
  };
};

export const configToInfo = (config: RuntimeConfig): RuntimeInfo => {
  const {instanceManagerType} = config;
  if (instanceManagerType === 'GCP') {
    return {
      type: RuntimeType.cloud,
    };
  }

  return {
    type: RuntimeType.local,
  };
};
