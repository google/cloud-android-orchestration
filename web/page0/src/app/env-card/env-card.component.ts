import { Component, Input } from '@angular/core';
import { Envrionment } from '../env-interface';

@Component({
  selector: 'app-env-card',
  templateUrl: './env-card.component.html',
  styleUrls: ['./env-card.component.scss'],
})
export class EnvCardComponent {
  @Input() env!: Envrionment;

  constructor() {}

  ngOnInit() {
    const status = this.env.status;

    if (status === 'running') {
      this.tooltip = 'Running';
      this.icon = 'check_circle';
      return;
    }

    if (status === 'stopping') {
      this.tooltip = 'Stopping';
      this.icon = 'stop_circle';
      return;
    }

    if (status === 'error') {
      this.tooltip = 'error';
      this.icon = 'error';
      return;
    }

    this.tooltip = 'Starting';
    this.icon = 'pending';
  }

  tooltip = '';
  icon = '';

  getColor() {
    return 'aliceblue';
  }

  goToPerGroupUI() {
    // TODO: Open per-group UI w/ safeurl
  }
}
