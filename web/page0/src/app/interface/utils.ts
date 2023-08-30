import {DeviceSetting} from 'src/app/interface/device-interface';
import {CVD} from 'src/app/interface/host-orchestrator.dto';

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
