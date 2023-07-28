import { Component, Input } from '@angular/core';
import { Router } from '@angular/router';
import { Runtime } from '../runtime-interface';

@Component({
  selector: 'app-runtime-card',
  templateUrl: './runtime-card.component.html',
  styleUrls: ['./runtime-card.component.scss']
})
export class RuntimeCardComponent {
  @Input() runtime: Runtime | null = null;
  panelOpenState = false

  status = "error"

  constructor(private router: Router) {}
  onClickAddHost() {
    this.router.navigate(
      ['/create-host'],
      { queryParams: { runtime: this.runtime?.alias }}
    )

  }
}
