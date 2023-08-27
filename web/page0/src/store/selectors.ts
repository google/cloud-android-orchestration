import {AppState} from './state';
import {runtimeToEnvList} from '../interface/utils';
import {EnvStatus} from 'src/app/env-interface';

// TODO: add starting & stopping envs here
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

export const envSelector = (state: AppState) => {
  const runningAndStoppingEnvs = state.runtimes
    .flatMap(runtime => runtimeToEnvList(runtime))
    .map(env => {
      if (
        !state.stoppingEnvs.find(
          stoppingEnv =>
            stoppingEnv.groupName === env.groupName &&
            stoppingEnv.hostUrl === env.hostUrl
        )
      ) {
        return env;
      }

      return {
        ...env,
        status: EnvStatus.stopping,
      };
    });

  return [...runningAndStoppingEnvs, ...state.startingEnvs];
};
