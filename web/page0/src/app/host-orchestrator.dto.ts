interface AndroidCIBuild {
  branch: string;
  build_id: string;
  target: string;
}

interface AndroidCIBuildSource {
  main_build?: AndroidCIBuild;
  kernel_build?: AndroidCIBuild;
  bootloader_build?: AndroidCIBuild;
  system_image_build?: AndroidCIBuild;
  credentials?: string;
}

export interface CVD {
  name: string;
  build_source: BuildSource;
  status: string;
  displays: string[];
  group_name?: string; // TODO: Not in current host orchestrator
}

export interface BuildSource {
  android_ci_build_source: AndroidCIBuildSource;
  // TODO: user build
}

export interface ListCVDsResponse {
  cvds: CVD[];
}

// TODO: Not in current host orchestrator from here

export interface Group {
  name: string;
  cvds: CVD[];
}

export interface ListGroupsResponse {
  groups: Group[];
}

export interface CreateGroupRequest {
  group: Group;
}

export interface CreateGroupResponse {
  group: Group;
}
