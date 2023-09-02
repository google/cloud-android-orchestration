export const defaultRuntimeSettings = [
  {
    alias: 'default',
    url: 'http://localhost:8071/api',
  },
];

export const placeholderRuntimeSetting = {
  alias: 'example',
  url: 'http://localhost:8071/api',
};

export const defaultEnvConfig: object = {
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

// TODO: default zone & host setting
