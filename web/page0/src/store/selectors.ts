import {DeviceSetting} from 'src/app/device-interface';
import {Environment, EnvStatus} from 'src/app/env-interface';
import {Host} from 'src/app/host-interface';
import {CVD} from 'src/app/host-orchestrator.dto';
import {Runtime} from 'src/app/runtime-interface';
import {AppState} from './state';

export const runtimeListSelector = (state: AppState) => state.runtimes;

export const hostListSelector = (state: AppState) =>
  state.runtimes.flatMap(runtime => runtime.hosts);

export const hostListSelectorFactory = (params: {
  runtimeAlias: string;
  zone?: string;
}) => {
  const {runtimeAlias, zone} = params;

  return (state: AppState) => {
    const runtime = state.runtimes.find(
      runtime => runtime.alias === runtimeAlias
    );

    if (!runtime) {
      return [];
    }

    if (!zone) {
      return runtime.hosts;
    }

    return runtime.hosts.filter(host => host.zone === zone);
  };
};

export const hostSelectorFactory = (params: {
  runtimeAlias: string;
  zone: string;
  name: string;
}) => {
  return (state: AppState) => {
    const {runtimeAlias, zone, name} = params;

    const runtime = state.runtimes.find(
      runtime => runtime.alias === runtimeAlias
    );
    if (!runtime) {
      return undefined;
    }

    return runtime.hosts.find(host => host.zone === zone && host.name === name);
  };
};

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

const runtimeToEnvList = (runtime: Runtime): Environment[] => {
  return runtime.hosts.flatMap(host => hostToEnvList(host));
};

export const envSelector = (state: AppState) => {
  return state.runtimes.flatMap(runtime => runtimeToEnvList(runtime));
};
