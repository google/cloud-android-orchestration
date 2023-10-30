export declare interface AndroidCIBuild {
  branch: string;
  build_id: string;
  target: string;
}

export declare interface AndroidCIBuildSource {
  main_build: AndroidCIBuild;
  // kernel_build?: AndroidCIBuild;
  // bootloader_build?: AndroidCIBuild;
  // system_image_build?: AndroidCIBuild;
  // credentials?: string;
}

export declare interface CVD {
  name: string;
  build_source: BuildSource;
  status: string;
  displays: string[];
  group_name?: string; // TODO: Not in current host orchestrator
}

export declare interface BuildSource {
  android_ci_build_source: AndroidCIBuildSource;
  // TODO: user build
}

export declare interface ListCVDsResponse {
  cvds: CVD[];
}

export declare interface Group {
  name: string;
  cvds: CVD[];
}

export declare interface CreateCVDRequest {
  env_config: object;
}
