import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { AppComponent } from './app.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientModule } from '@angular/common/http';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDividerModule } from '@angular/material/divider';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatListModule } from '@angular/material/list';
import { MatExpansionModule } from '@angular/material/expansion';

import { RouterModule } from '@angular/router';

import { EnvListViewComponent } from './env-list-view/env-list-view.component';
import { ActiveEnvPaneComponent } from './active-env-pane/active-env-pane.component';
import { EnvCardComponent } from './env-card/env-card.component';
import { CreateEnvViewComponent } from './create-env-view/create-env-view.component';
import { NgIf } from '@angular/common';
import { RegisterRuntimeViewComponent } from './register-runtime-view/register-runtime-view.component';
import { CreateHostViewComponent } from './create-host-view/create-host-view.component';
import { ListRuntimeViewComponent } from './list-runtime-view/list-runtime-view.component';
import { RuntimeCardComponent } from './runtime-card/runtime-card.component';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { DeviceFormComponent } from './device-form/device-form.component';
import { SafeUrlPipe } from './safe-url.pipe';

@NgModule({
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
    RouterModule.forRoot([
      { path: 'create-env', component: CreateEnvViewComponent },
      { path: '', component: EnvListViewComponent },
      { path: 'list-runtime', component: ListRuntimeViewComponent },
      { path: 'create-host', component: CreateHostViewComponent },
      { path: 'register-runtime', component: RegisterRuntimeViewComponent },
    ]),
  ],
  providers: [],
  bootstrap: [AppComponent],
})
export class AppModule {}
