import {NgIf} from '@angular/common';
import {HttpClientModule} from '@angular/common/http';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import {ComponentFixtureAutoDetect} from '@angular/core/testing';
import {ReactiveFormsModule, FormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {MatDividerModule} from '@angular/material/divider';
import {MatExpansionModule} from '@angular/material/expansion';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatListModule} from '@angular/material/list';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {MatSelectModule} from '@angular/material/select';
import {MatSidenavModule} from '@angular/material/sidenav';
import {MatSlideToggleModule} from '@angular/material/slide-toggle';
import {MatSnackBarModule} from '@angular/material/snack-bar';
import {MatToolbarModule} from '@angular/material/toolbar';
import {MatTooltipModule} from '@angular/material/tooltip';
import {BrowserModule} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
import {RouterModule} from '@angular/router';
import {ActiveEnvPaneComponent} from './active-env-pane/active-env-pane.component';
import {AppComponent} from './app.component';
import {CreateEnvViewComponent} from './create-env-view/create-env-view.component';
import {CreateHostViewComponent} from './create-host-view/create-host-view.component';
import {DeviceFormComponent} from './device-form/device-form.component';
import {EnvCardComponent} from './env-card/env-card.component';
import {EnvListViewComponent} from './env-list-view/env-list-view.component';
import {ListRuntimeViewComponent} from './list-runtime-view/list-runtime-view.component';
import {RegisterRuntimeViewComponent} from './register-runtime-view/register-runtime-view.component';
import {RuntimeCardComponent} from './runtime-card/runtime-card.component';
import {SafeUrlPipe} from './safe-url.pipe';

export const modules = {
  declarations: [
    AppComponent,
    EnvListViewComponent,
    ActiveEnvPaneComponent,
    EnvCardComponent,
    CreateEnvViewComponent,
    RegisterRuntimeViewComponent,
    CreateHostViewComponent,
    ListRuntimeViewComponent,
    RuntimeCardComponent,
    DeviceFormComponent,
    SafeUrlPipe,
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    HttpClientTestingModule,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatIconModule,
    MatSidenavModule,
    MatSlideToggleModule,
    MatToolbarModule,
    MatTooltipModule,
    MatDividerModule,
    MatInputModule,
    MatFormFieldModule,
    MatSelectModule,
    MatListModule,
    MatExpansionModule,
    MatProgressBarModule,
    NgIf,
    ReactiveFormsModule,
    FormsModule,
    HttpClientModule,
    MatSnackBarModule,
    RouterModule,
  ],
  providers: [{provide: ComponentFixtureAutoDetect, useValue: true}],
};
