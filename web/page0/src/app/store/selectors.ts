import {AppState} from './state';
import {EnvStatus} from 'src/app/interface/env-interface';
import {Runtime, RuntimeStatus} from '../interface/runtime-interface';
import {HostStatus} from '../interface/host-interface';
import {isHostCreateWait, isHostDeleteWait} from '../interface/wait-interface';
import {RuntimeCard} from '../interface/component-interface';

export const runtimeListSelector = (state: AppState) => state.runtimes;

export const validRuntimeListSelector = (state: AppState) =>
  state.runtimes.filter(runtime => runtime.status === RuntimeStatus.valid);

export const runtimeCardSelectorFactory =
  (alias: string | undefined) =>
  (state: AppState): RuntimeCard | undefined => {
    if (!alias) {
      return undefined;
    }

    const runtime = runtimeSelectorFactory({alias})(state);
    if (!runtime) {
      return undefined;
    }

    const hostCreateRequests = Object.values(state.waits)
      .filter(isHostCreateWait)
      .filter(wait => wait.metadata.runtimeAlias === alias);

    const hostDeleteRequests = Object.values(state.waits).filter(
      isHostDeleteWait
    );

    return {
      alias: runtime.alias,
      url: runtime.url,
      type: runtime.type,
      hosts: [
        ...state.hosts.map(host => {
          const isStopping = hostDeleteRequests.find(
            req => req.metadata.hostUrl === host.url
          );
          if (!isStopping) {
            return {
              ...host,
              envs: state.envs.filter(env => env.hostUrl === host.url),
              status: HostStatus.running,
            };
          }
          return {
            ...host,
            envs: [],
            status: HostStatus.stopping,
          };
        }),
        ...hostCreateRequests.map(wait => ({
          name: 'New host',
          zone: wait.metadata.zone,
          runtime: alias,
          envs: [],
          status: HostStatus.starting,
        })),
      ],
      status: runtime.status,
    };
  };

export const hostSelectorFactory = (params: {
  runtimeAlias: string;
  zone: string;
  name: string;
}) => {
  return (state: AppState) => {
    const {runtimeAlias, zone, name} = params;

    const host = state.hosts.find(host => {
      return (
        host.runtime === runtimeAlias &&
        host.zone === zone &&
        host.name === name
      );
    });

    if (!host) {
      return undefined;
    }

    return host;
  };
};

export const hostListSelectorFactory = (params: {
  runtimeAlias: string;
  zone?: string;
}) => {
  const {runtimeAlias, zone} = params;

  return (state: AppState) => {
    return state.hosts.filter(host => {
      if (host.runtime !== runtimeAlias) {
        return false;
      }

      if (zone && host.zone !== zone) {
        return false;
      }

      return true;
    });
  };
};

export const envCardListSelector = (state: AppState) => {
  // TODO: get starting & stopping envs from state.waits
  return state.envs;
};

export const runtimesLoadStatusSelector = (state: AppState) => {
  return state.runtimesLoadStatus;
};

export const runtimeSelectorFactory = (params: {
  alias: string;
}): ((state: AppState) => Runtime | undefined) => {
  return (state: AppState) => {
    return state.runtimes.find(runtime => runtime.alias === params.alias);
  };
};
