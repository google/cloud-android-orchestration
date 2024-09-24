import {Component, Input} from '@angular/core';
import {FormGroup} from '@angular/forms';
import {EnvFormService} from '../env-form.service';

@Component({
  standalone: false,
  selector: 'app-device-form',
  templateUrl: './device-form.component.html',
  styleUrls: ['./device-form.component.scss'],
})
export class DeviceFormComponent {
  @Input() form!: FormGroup;
  @Input() idx!: number;

  constructor(private envFormService: EnvFormService) {}

  onClickDeleteDevice() {
    this.envFormService.deleteDevice(this.idx);
  }

  onClickDuplicateDevice() {
    this.envFormService.duplicateDevice(this.idx);
  }
}
