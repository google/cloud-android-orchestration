import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {
  CreateHostRequest,
  ListHostsResponse,
  ListZonesResponse,
  Operation,
  RuntimeConfig,
} from 'src/app/interface/cloud-orchestrator.dto';
import {ListCVDsResponse} from 'src/app/interface/host-orchestrator.dto';

@Injectable({
  providedIn: 'root',
})
export class ApiService {
  constructor(private httpClient: HttpClient) {}

  // Global Routes
  getRuntimeConfig(runtimeUrl: string) {
    return this.httpClient.get<RuntimeConfig>(`${runtimeUrl}/v1/config`);
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
    return this.httpClient.delete<Operation>(`${hostUrl}`);
  }

  // Host Orchestrator Proxy Routes
  listGroups(hostUrl: string) {
    return this.httpClient.get<string[]>(`${hostUrl}/groups`);
  }

  createGroup(hostUrl: string, config: object) {
    return this.httpClient.post<Operation>(`${hostUrl}/cvds`, {
      env_config: config,
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

  wait<T>(waitUrl: string) {
    return this.httpClient.post<T>(`${waitUrl}/:wait`, {});
  }
}
