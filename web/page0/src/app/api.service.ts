import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import {
  CreateHostRequest,
  ListHostsResponse,
  ListZonesResponse,
  Operation,
  RuntimeResponse,
} from './cloud-orchestrator.dto';
import {
  ListCVDsResponse,
  ListGroupsResponse,
  CreateGroupRequest,
  CreateGroupResponse,
} from './host-orchestrator.dto';

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
    return this.httpClient.get<ListGroupsResponse>(`${hostUrl}/groups`);
  }

  createGroup(hostUrl: string, createGroupRequest: CreateGroupRequest) {
    return this.httpClient.post<CreateGroupResponse>(
      `${hostUrl}/cvds`,
      createGroupRequest
    );
  }

  deleteGroup(hostUrl: string, groupName: string) {
    return this.httpClient.delete(`${hostUrl}/groups/${groupName}`);
  }

  listCvds(hostUrl: string) {
    return this.httpClient.get<ListCVDsResponse>(`${hostUrl}/cvds`);
  }
}
