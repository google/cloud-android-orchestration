import {RequestMatch} from '@angular/common/http/testing';
import {Host} from 'src/app/host-interface';
import {Runtime} from 'src/app/runtime-interface';

export interface MockCloudOrchestrator {
  alias: string;
  type?: 'local' | 'on-premise' | 'cloud';
  url: string;
  zones: string[];
  hosts: Host[];
}

interface MockApi {
  params: RequestMatch;
  data: object;
  opts?: object;
}

export const deriveApis = (mockCloudOrchestrator: Runtime): MockApi[] => {
  const {url, zones, hosts, type} = mockCloudOrchestrator;

  if (!zones) {
    return [
      {
        params: {
          method: 'GET',
          url: `${url}/info`,
        },
        data: {},
        opts: {
          status: 500,
          statusText: 'Internal server error',
        },
      },
    ];
  }

  return [
    {
      params: {
        method: 'GET',
        url: `${url}/info`,
      },
      data: {
        type,
      },
    },

    {
      params: {
        method: 'GET',
        url: `${url}/v1/zones`,
      },
      data: {
        items: zones.map(zone => ({
          name: zone,
        })),
      },
    },

    ...zones.map(zone => ({
      params: {
        method: 'GET',
        url: `${url}/v1/zones/${zone}/hosts`,
      },
      data: hosts.filter(host => host.zone === zone),
    })),

    ...hosts.map(host => ({
      params: {
        method: 'GET',
        url: `${url}/v1/zones/${host.zone}/hosts/${host.name}/groups`,
      },
      data: host.groups,
    })),
  ];
};

export class MockLocalStorage {
  constructor(store: any) {
    this.store = store;
  }

  store: any;

  getItem(key: string): string {
    return JSON.stringify(this.store[key]);
  }
  setItem(key: string, value: string) {
    this.store[key] = value;
  }
  removeItem(key: string) {
    delete this.store[key];
  }
  clear() {
    this.store = {};
  }
}
