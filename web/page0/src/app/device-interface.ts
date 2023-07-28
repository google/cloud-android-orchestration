interface BaseDevice {
  device_id: string
  fingerprint: string
}

interface LocalDevice extends BaseDevice {
  upload_dir: string
}

interface RemoteDevice extends BaseDevice {
  target: string
  build_id: string
}

export type Device = LocalDevice | RemoteDevice