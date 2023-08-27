import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {
  CreateHostRequest,
  ListHostsResponse,
  ListZonesResponse,
  Operation,
  RuntimeInfo,
} from './cloud-orchestrator.dto';
import {ListCVDsResponse, CreateGroupRequest} from './host-orchestrator.dto';

@Injectable({
  providedIn: 'root',
})
export class ApiService {
  constructor(private httpClient: HttpClient) {}

  // Global Routes
  getRuntimeInfo(runtimeUrl: string) {
    return this.httpClient.get<RuntimeInfo>(`${runtimeUrl}/v1/info`);
  }

  listZones(runtimeUrl: string) {
    return this.httpClient.get<ListZonesResponse>(`${runtimeUrl}/v1/zones`);
  }

  // Instance Manager Routes
  createHost(
    runtimeUrl: string,
    zone: string,
    createHostRequest: CreateHostRequest
  ) {
    return this.httpClient.post<Operation>(
      `${runtimeUrl}/v1/zones/${zone}/hosts`,
      createHostRequest
    );
  }

  listHosts(runtimeUrl: string, zone: string) {
    return this.httpClient.get<ListHostsResponse>(
      `${runtimeUrl}/v1/zones/${zone}/hosts`
    );
  }

  deleteHost(hostUrl: string) {
    return this.httpClient.delete<void>(`${hostUrl}`);
  }

  // Host Orchestrator Proxy Routes
  listGroups(hostUrl: string) {
    return this.httpClient.get<string[]>(`${hostUrl}/groups`);
  }

  createGroup(hostUrl: string, createGroupRequest: CreateGroupRequest) {
    return this.httpClient.post<void>(`${hostUrl}/cvds`, {
      // TODO: use data from createGroupRequest for cvd
      group_name: createGroupRequest.group_name,
      cvd: {
        name: '',
        build_source: {
          android_ci_build_source: {
            branch: 'aosp-main',
            build_id: '10678986',
            target: 'aosp_cf_x86_64_phone-trunk_staging-userdebug',
          },
        },
        status: '',
        displays: [],
      },
      instance_names: createGroupRequest.instance_names,
    });
  }

  deleteGroup(hostUrl: string, groupName: string) {
    return this.httpClient.delete(`${hostUrl}/groups/${groupName}`);
  }

  listDevicesByGroup(hostUrl: string, groupName: string) {
    return this.httpClient.get<{device_id: string; group_id: string}[]>(
      `${hostUrl}/devices?groupId=${groupName}`
    );
  }

  listCvds(hostUrl: string) {
    return this.httpClient.get<ListCVDsResponse>(`${hostUrl}/cvds`);
  }
}
