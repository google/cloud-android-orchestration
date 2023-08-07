import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import {
  CreateHostRequest,
  ListHostsResponse,
  ListZonesResponse,
  Operation,
  RuntimeResponse,
} from './cloud-orchestrator.dto';
import { ListGroupsResponse } from './host-orchestrator.dto';

@Injectable({
  providedIn: 'root',
})
export class ApiService {
  constructor(private httpClient: HttpClient) {}

  // Global Routes
  getRuntimeInfo(runtimeUrl: string) {
    return this.httpClient.get<RuntimeResponse>(`${runtimeUrl}/info`);
  }

  listZones(runtimeUrl: string) {
    return this.httpClient.get<ListZonesResponse>(`${runtimeUrl}/zones`);
  }

  // Instance Manager Routes
  createHost(
    runtimeUrl: string,
    zone: string,
    createHostInstance: CreateHostRequest
  ) {
    return this.httpClient.post<Operation>(
      `${runtimeUrl}/v1/zones/${zone}/hosts`,
      createHostInstance
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
    return this.httpClient.get<ListGroupsResponse>(`${hostUrl}/groups`);
  }

  createGroup() {}

  deleteGroup(hostUrl: string, groupName: string) {}

  listCvds(hostUrl: string, groupName: string) {}
}
