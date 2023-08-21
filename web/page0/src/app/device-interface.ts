export interface DeviceSetting {
  deviceId: string;
  branch: string;
  target: string;
  buildId: string;
}

export interface GroupForm {
  groupName: string;
  devices: DeviceSetting[];
}
