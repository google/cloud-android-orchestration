import { Component, Input } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router } from '@angular/router';
import { BehaviorSubject, Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { Host } from '../host-interface';
import { HostService } from '../host.service';
import { Runtime } from '../runtime-interface';
import { RuntimeService } from '../runtime.service';

@Component({
  selector: 'app-runtime-card',
  templateUrl: './runtime-card.component.html',
  styleUrls: ['./runtime-card.component.scss'],
})
export class RuntimeCardComponent {
  @Input() runtime: Runtime | null = null;

  hosts$ = new BehaviorSubject<Host[]>([]);

  private ngUnsubscribe = new Subject<void>();

  constructor(
    private router: Router,
    private runtimeService: RuntimeService,
    private hostService: HostService,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit() {
    if (!this.runtime) {
      return;
    }

    this.hostService
      .getHosts(this.runtime.alias)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe((hosts) => this.hosts$.next(hosts));
  }

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  onClickAddHost() {
    this.router.navigate(['/create-host'], {
      queryParams: { runtime: this.runtime?.alias },
    });
  }

  onClickUnregister(alias: string | undefined) {
    if (!alias) {
      return;
    }

    this.runtimeService.unregisterRuntime(alias);
  }

  onClickDeleteHost(host: Host) {
    this.snackBar.open(
      `Start to delete host ${host.name} (url: ${host.url})`,
      'dismiss'
    );
    this.hostService
      .deleteHost(host.url)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe({
        next: () => {
          this.snackBar.dismiss();
        },
        error: (error) => {
          this.snackBar.open(
            `Failed to delete host ${host.url} (error: ${error.message})`,
            'dismiss'
          );
        },
      });
  }
}
