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
