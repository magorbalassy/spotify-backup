import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIcon } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

@Component({
  selector: 'app-config',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule
  ],
  templateUrl: './config.html',
  styleUrls: ['./config.css']
})
export class Config {
  clientId = signal('');
  clientSecret = signal('');
  saving = signal(false);
  success = signal(false);
  error = signal<string | null>(null);

  saveCredentials() {
    if (!this.clientId() || !this.clientSecret()) {
      this.error.set('Please fill in both Client ID and Client Secret');
      return;
    }

    this.saving.set(true);
    this.error.set(null);
    this.success.set(false);

    // TODO: Call backend service to save credentials
    // For now, just simulate success
    setTimeout(() => {
      this.saving.set(false);
      this.success.set(true);
      console.log('Credentials saved:', {
        clientId: this.clientId(),
        clientSecret: this.clientSecret()
      });
    }, 1000);
  }

  clearForm() {
    this.clientId.set('');
    this.clientSecret.set('');
    this.error.set(null);
    this.success.set(false);
  }
}