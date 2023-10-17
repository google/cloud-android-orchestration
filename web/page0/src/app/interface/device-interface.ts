export interface DeviceSetting {
  deviceId: string;
  target: string;
  branch_or_buildId: string;
}

export interface GroupForm {
  groupName: string;
  devices: DeviceSetting[];
}
