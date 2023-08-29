import {Component, Input} from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';
import {Router} from '@angular/router';
import {runtimeCardSelectorFactory} from 'src/app/store/selectors';
import {Store} from 'src/app/store/store';
import {Host, HostStatus} from 'src/app/interface/host-interface';
import {HostService} from '../host.service';
import {RuntimeService} from '../runtime.service';

@Component({
  selector: 'app-runtime-card',
  templateUrl: './runtime-card.component.html',
  styleUrls: ['./runtime-card.component.scss'],
})
export class RuntimeCardComponent {
  @Input() runtimeAlias = '';

  getRuntimeCard = (alias: string) =>
    this.store.select(runtimeCardSelectorFactory(alias));

  constructor(
    private router: Router,
    private runtimeService: RuntimeService,
    private hostService: HostService,
    private snackBar: MatSnackBar,
    private store: Store
  ) {}

  onClickAddHost() {
    this.router.navigate(['/create-host'], {
      queryParams: {runtime: this.runtimeAlias},
    });
  }

  onClickUnregister(alias: string | undefined) {
    if (!alias) {
      return;
    }

    this.runtimeService.unregisterRuntime(alias);
  }

  onClickDeleteHost(host: Host) {
    if (host.status !== HostStatus.running || !host.url) {
      this.snackBar.open('Cannot delete non-running host', 'dismiss');
      return;
    }

    this.snackBar.open(
      `Start to delete host ${host.name} (url: ${host.url})`,
      'dismiss'
    );

    this.hostService.deleteHost(host.url!).subscribe({
      next: () => {
        this.snackBar.dismiss();
      },
      error: error => {
        this.snackBar.open(
          `Failed to delete host ${host.url} (error: ${error.message})`,
          'dismiss'
        );
      },
    });
  }

  isRunning(host: Host) {
    return host.status === HostStatus.running;
  }
}
