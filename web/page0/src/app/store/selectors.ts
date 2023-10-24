import {AppState} from './state';
import {EnvStatus} from 'src/app/interface/env-interface';
import {Runtime, RuntimeStatus} from '../interface/runtime-interface';
import {HostStatus} from '../interface/host-interface';
import {
  isEnvAutoHostCreateWait,
  isEnvCreateWait,
  isHostCreateWait,
  isHostDeleteWait,
} from '../interface/wait-interface';
import {RuntimeCard} from '../interface/component-interface';

export function runtimeListSelector(state: AppState) {
  return state.runtimes;
}

export function validRuntimeListSelector(state: AppState) {
  return state.runtimes.filter(runtime => runtime.status === RuntimeStatus.valid);
}

export function runtimeCardSelectorFactory(alias: string | undefined) {
  return (state: AppState): RuntimeCard | undefined => {
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

    const autoHostCreateRequests = Object.values(state.waits)
      .filter(isEnvAutoHostCreateWait)
      .filter(wait => wait.metadata.runtimeAlias === alias);

    const hostDeleteRequests = Object.values(state.waits).filter(
      isHostDeleteWait
    );

    const allEnvs = allEnvListSelector(state).filter(
      env => env.runtimeAlias === alias
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
              envs: allEnvs.filter(env => env.hostUrl === host.url),
              status: HostStatus.running,
            };
          }
          return {
            ...host,
            envs: [],
            status: HostStatus.stopping,
          };
        }),
        ...[...hostCreateRequests, ...autoHostCreateRequests].map(wait => ({
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
}

export function hostSelectorFactory (params: {
  runtimeAlias: string;
  zone: string;
  name: string;
}) {
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
}

export function hostListSelectorFactory(params: {
  runtimeAlias: string;
  zone?: string;
}) {
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
}

export function envCardListSelector(state: AppState) {
  return allEnvListSelector(state);
}

function allEnvListSelector(state: AppState) {
  const envCreateRequests = Object.values(state.waits).filter(isEnvCreateWait);

  const envAutoHostCreateRequests = Object.values(state.waits).filter(
    isEnvAutoHostCreateWait
  );

  const hostDeleteRequests = Object.values(state.waits).filter(
    isHostDeleteWait
  );

  const startingEnvs = [
    ...envCreateRequests.map(req => {
      const {groupName, hostUrl, runtimeAlias, devices} = req.metadata;

      return {
        groupName,
        hostUrl,
        devices,
        runtimeAlias,
        status: EnvStatus.starting,
      };
    }),

    ...envAutoHostCreateRequests.map(req => {
      const {groupName, runtimeAlias, devices} = req.metadata;

      return {
        groupName,
        hostUrl: 'unknown',
        devices,
        runtimeAlias,
        status: EnvStatus.starting,
      };
    }),
  ];

  return [
    ...startingEnvs.filter(
      env =>
        !state.envs.find(
          runningEnv =>
            runningEnv.groupName === env.groupName &&
            runningEnv.hostUrl === env.hostUrl
        )
    ),
    ...state.envs.map(env => {
      if (
        hostDeleteRequests.find(req => req.metadata.hostUrl === env.hostUrl)
      ) {
        return {
          ...env,
          status: EnvStatus.stopping,
        };
      }

      return env;
    }),
  ];
}

export function runtimesLoadStatusSelector(state: AppState) {
  return state.runtimesLoadStatus;
}

export function runtimeSelectorFactory(params: {
  alias: string;
}): ((state: AppState) => Runtime | undefined) {
  return (state: AppState) => {
    return state.runtimes.find(runtime => runtime.alias === params.alias);
  };
}
