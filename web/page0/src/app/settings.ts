import {HostInstance} from './interface/cloud-orchestrator.dto';

export const DEFAULT_RUNTIME_SETTINGS = [];

export const PLACEHOLDER_RUNTIME_SETTING = {
  alias: 'example-runtime-setting',
  url: 'https://example-runtime-setting.com/',
};

export const DEFAULT_ENV_CONFIG: object = {
  // common: {
  //   group_name: 'simulated_home',
  // },
  instances: [
    {
      name: 'my_phone',
      disk: {
        default_build:
          '@ab/aosp-main/aosp_cf_x86_64_phone-trunk_staging-userdebug',
      },
    },
    // {
    //   name: 'my_watch',
    //   disk: {
    //     default_build: '@ab/git_main/cf_gwear_x86-trunk_staging-userdebug',
    //   },
    // },
  ],
};

export const DEFAULT_ZONE = 'us-east1-b';

export const DEFAULT_HOST_SETTING: HostInstance = {
  gcp: {
    machine_type: 'n1-standard-4',
    min_cpu_platform: 'Intel Skylake',
  },
};
