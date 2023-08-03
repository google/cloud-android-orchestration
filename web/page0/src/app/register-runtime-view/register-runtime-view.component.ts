import { Component } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { first } from 'rxjs';
import { RuntimeService } from '../runtime.service';
import { MatSnackBar } from '@angular/material/snack-bar';

@Component({
  selector: 'app-register-runtime-view',
  templateUrl: './register-runtime-view.component.html',
  styleUrls: ['./register-runtime-view.component.scss'],
})
export class RegisterRuntimeViewComponent {
  constructor(
    private runtimeService: RuntimeService,
    private formBuilder: FormBuilder,
    private router: Router,
    private snackBar: MatSnackBar
  ) {}

  runtimes$ = this.runtimeService.getRuntimes();
  loading = false;

  runtimeForm = this.formBuilder.group({
    url: ['http://localhost:3000', Validators.required],
    alias: ['test', Validators.required],
  });

  onSubmit() {
    const url = this.runtimeForm.value.url;
    const alias = this.runtimeForm.value.alias;

    if (!url || !alias) {
      return;
    }

    this.loading = true;
    this.runtimeService
      .verifyRuntime(url, alias)
      .pipe(first())
      .subscribe({
        next: (runtime) => {
          this.runtimeService.registerRuntime(runtime);
          this.router.navigate(['/list-runtime']);
          this.snackBar.dismiss();
          this.loading = false;
        },
        error: (error) => {
          this.snackBar.open(error.message);
          this.loading = false;
        },
      });
  }
}
