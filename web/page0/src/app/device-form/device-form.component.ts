import { Component, Input } from '@angular/core';
import { FormGroup } from '@angular/forms';
import { DeviceFormService } from '../device-form.service';

@Component({
  selector: 'app-device-form',
  templateUrl: './device-form.component.html',
  styleUrls: ['./device-form.component.scss'],
})
export class DeviceFormComponent {
  @Input() form!: FormGroup;
  @Input() idx!: number;

  constructor(private deviceFormService: DeviceFormService) {}

  onClickDeleteDevice() {
    this.deviceFormService.deleteDevice(this.idx);
  }

  onClickDuplicateDevice() {
    this.deviceFormService.duplicateDevice(this.idx);
  }
}
