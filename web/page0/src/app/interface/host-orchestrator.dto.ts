interface AndroidCIBuild {
  branch: string;
  build_id: string;
  target: string;
}

interface AndroidCIBuildSource {
  main_build: AndroidCIBuild;
  // kernel_build?: AndroidCIBuild;
  // bootloader_build?: AndroidCIBuild;
  // system_image_build?: AndroidCIBuild;
  // credentials?: string;
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

export interface Group {
  name: string;
  cvds: CVD[];
}

export interface CreateGroupRequest {
  group_name: string;
  cvd: CVD;
  instance_names: string[];
}
