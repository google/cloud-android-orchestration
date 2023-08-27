import {DeviceSetting} from 'src/app/device-interface';
import {Environment, EnvStatus} from 'src/app/env-interface';
import {Host} from 'src/app/host-interface';
import {CVD, Group} from 'src/app/host-orchestrator.dto';
import {Runtime} from 'src/app/runtime-interface';

const cvdToDevice = (cvd: CVD): DeviceSetting => {
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

const hostToEnvList = (host: Host): Environment[] => {
  return host.groups.flatMap(group => ({
    runtimeAlias: host.runtime,
    hostUrl: host.url,
    groupName: group.name,
    devices: group.cvds.map(cvd => cvdToDevice(cvd)),
    status: EnvStatus.running,
  }));
};

export const runtimeToEnvList = (runtime: Runtime): Environment[] => {
  return runtime.hosts.flatMap(host => hostToEnvList(host));
};
