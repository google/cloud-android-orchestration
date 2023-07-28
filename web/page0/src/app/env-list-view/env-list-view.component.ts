import { HttpClient } from '@angular/common/http';
import { Component } from '@angular/core';

@Component({
  selector: 'app-env-list-view',
  templateUrl: './env-list-view.component.html',
  styleUrls: ['./env-list-view.component.scss'],
})
export class EnvListViewComponent {
  constructor(private httpClient: HttpClient) {}

  ngOnInit() {}
}
