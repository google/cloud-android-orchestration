import {Component, Input} from '@angular/core';
import {Environment, EnvStatus} from '../interface/env-interface';
import {HostService} from '../host.service';
import {MatSnackBar} from '@angular/material/snack-bar';

const tooltips = {
  [EnvStatus.starting]: 'Starting',
  [EnvStatus.running]: 'Running',
  [EnvStatus.stopping]: 'Stopping',
  [EnvStatus.error]: 'Error',
};

const icons = {
  [EnvStatus.starting]: 'pending',
  [EnvStatus.running]: 'check_circle',
  [EnvStatus.stopping]: 'stop_circle',
  [EnvStatus.error]: 'error',
};

@Component({
  selector: 'app-env-card',
  templateUrl: './env-card.component.html',
  styleUrls: ['./env-card.component.scss'],
})
export class EnvCardComponent {
  @Input() env!: Environment;

  constructor(
    private hostService: HostService,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit() {}

  getCardSetting() {
    const status = this.env.status;
    return {
      tooltip: tooltips[status],
      icon: icons[status],
      backgroundColor: 'aliceblue',
    };
  }

  isRunning() {
    return this.env.status === EnvStatus.running;
  }

  onClickGoto() {
    const {hostUrl, groupName} = this.env;
    // TODO: use safeurl
    window.open(`${hostUrl}/?groupId=${groupName}`);
  }

  onClickDelete() {
    this.hostService.deleteHost(this.env.hostUrl).subscribe({
      next: () => {
        this.snackBar.dismiss();
      },
      error: error => {
        this.snackBar.open(
          `Failed to delete host ${this.env.hostUrl} (error: ${error.message})`,
          'Dismiss'
        );
      },
    });
  }
}
