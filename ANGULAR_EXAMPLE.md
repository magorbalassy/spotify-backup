# Angular Integration Example

## Service Example

```typescript
// spotify-auth.service.ts
import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, interval } from 'rxjs';
import { switchMap, takeWhile } from 'rxjs/operators';

interface StatusResponse {
  hasToken: boolean;
  hasClientId: boolean;
  needsSetup: boolean;
  message: string;
}

interface AuthSetupRequest {
  clientId: string;
  clientSecret: string;
}

interface AuthSetupResponse {
  success: boolean;
  message: string;
  authUrl?: string;
}

@Injectable({
  providedIn: 'root'
})
export class SpotifyAuthService {
  private apiUrl = 'http://localhost:8080/api';
  private authWindow: Window | null = null;

  constructor(private http: HttpClient) {}

  getStatus(): Observable<StatusResponse> {
    return this.http.get<StatusResponse>(`${this.apiUrl}/status`);
  }

  setupAuth(clientId: string, clientSecret: string): Observable<AuthSetupResponse> {
    return this.http.post<AuthSetupResponse>(`${this.apiUrl}/auth/setup`, {
      clientId,
      clientSecret
    });
  }

  startAuth(): Observable<{ authUrl: string; message: string }> {
    return this.http.post<{ authUrl: string; message: string }>(
      `${this.apiUrl}/auth/start`,
      {}
    );
  }

  openAuthWindow(authUrl: string): void {
    const width = 600;
    const height = 700;
    const left = window.screenX + (window.outerWidth - width) / 2;
    const top = window.screenY + (window.outerHeight - height) / 2;
    
    this.authWindow = window.open(
      authUrl,
      'Spotify Authorization',
      `width=${width},height=${height},left=${left},top=${top}`
    );
  }

  pollForToken(): Observable<StatusResponse> {
    return interval(2000).pipe(
      switchMap(() => this.getStatus()),
      takeWhile(status => !status.hasToken, true)
    );
  }
}
```

## Component Example

```typescript
// auth.component.ts
import { Component, OnInit } from '@angular/core';
import { SpotifyAuthService } from './spotify-auth.service';

@Component({
  selector: 'app-auth',
  templateUrl: './auth.component.html'
})
export class AuthComponent implements OnInit {
  needsSetup = false;
  hasClientId = false;
  hasToken = false;
  message = '';
  clientId = '';
  clientSecret = '';
  loading = false;

  constructor(private authService: SpotifyAuthService) {}

  ngOnInit() {
    this.checkStatus();
  }

  checkStatus() {
    this.loading = true;
    this.authService.getStatus().subscribe({
      next: (status) => {
        this.needsSetup = status.needsSetup;
        this.hasClientId = status.hasClientId;
        this.hasToken = status.hasToken;
        this.message = status.message;
        this.loading = false;
      },
      error: (err) => {
        console.error('Failed to check status:', err);
        this.loading = false;
      }
    });
  }

  submitCredentials() {
    if (!this.clientId || !this.clientSecret) {
      alert('Please enter both Client ID and Client Secret');
      return;
    }

    this.loading = true;
    this.authService.setupAuth(this.clientId, this.clientSecret).subscribe({
      next: (response) => {
        if (response.success && response.authUrl) {
          this.authService.openAuthWindow(response.authUrl);
          this.startPolling();
        }
        this.loading = false;
      },
      error: (err) => {
        console.error('Failed to setup auth:', err);
        alert('Failed to setup authentication');
        this.loading = false;
      }
    });
  }

  authenticate() {
    this.loading = true;
    this.authService.startAuth().subscribe({
      next: (response) => {
        this.authService.openAuthWindow(response.authUrl);
        this.startPolling();
        this.loading = false;
      },
      error: (err) => {
        console.error('Failed to start auth:', err);
        this.loading = false;
      }
    });
  }

  private startPolling() {
    this.authService.pollForToken().subscribe({
      next: (status) => {
        if (status.hasToken) {
          this.hasToken = true;
          this.message = status.message;
          alert('Authentication successful!');
        }
      },
      error: (err) => {
        console.error('Polling error:', err);
      }
    });
  }
}
```

## Template Example

```html
<!-- auth.component.html -->
<div class="auth-container">
  <h1>Spotify Backup Authentication</h1>
  
  <div class="status-message">
    <p>{{ message }}</p>
  </div>

  <!-- State 1: Need to setup credentials -->
  <div *ngIf="needsSetup && !hasClientId" class="setup-form">
    <h2>Setup Spotify Credentials</h2>
    <p>To get started, you need to provide your Spotify API credentials.</p>
    <p><a href="https://developer.spotify.com/dashboard" target="_blank">Get credentials from Spotify Dashboard</a></p>
    
    <form (ngSubmit)="submitCredentials()">
      <div class="form-group">
        <label for="clientId">Client ID:</label>
        <input 
          type="text" 
          id="clientId" 
          [(ngModel)]="clientId" 
          name="clientId"
          required
          [disabled]="loading"
        />
      </div>
      
      <div class="form-group">
        <label for="clientSecret">Client Secret:</label>
        <input 
          type="password" 
          id="clientSecret" 
          [(ngModel)]="clientSecret" 
          name="clientSecret"
          required
          [disabled]="loading"
        />
      </div>
      
      <button type="submit" [disabled]="loading">
        {{ loading ? 'Setting up...' : 'Setup & Authenticate' }}
      </button>
    </form>
  </div>

  <!-- State 2: Has credentials, need to authenticate -->
  <div *ngIf="!needsSetup && hasClientId && !hasToken" class="auth-prompt">
    <h2>Ready to Authenticate</h2>
    <p>Your credentials are configured. Click the button below to authenticate with Spotify.</p>
    <button (click)="authenticate()" [disabled]="loading">
      {{ loading ? 'Opening...' : 'Authenticate with Spotify' }}
    </button>
  </div>

  <!-- State 3: Authenticated -->
  <div *ngIf="hasToken" class="auth-success">
    <h2>✓ Authentication Successful</h2>
    <p>You are now authenticated and ready to backup your playlists.</p>
    <button routerLink="/backup">Go to Backup</button>
  </div>

  <div *ngIf="loading" class="loading-spinner">
    <p>Loading...</p>
  </div>
</div>
```

## CSS Example

```css
/* auth.component.css */
.auth-container {
  max-width: 600px;
  margin: 50px auto;
  padding: 20px;
  background: #f5f5f5;
  border-radius: 8px;
}

h1 {
  text-align: center;
  color: #1db954; /* Spotify green */
}

.status-message {
  background: white;
  padding: 15px;
  border-radius: 4px;
  margin-bottom: 20px;
  text-align: center;
}

.setup-form, .auth-prompt, .auth-success {
  background: white;
  padding: 20px;
  border-radius: 4px;
}

.form-group {
  margin-bottom: 15px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
}

.form-group input {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  box-sizing: border-box;
}

button {
  background: #1db954;
  color: white;
  padding: 10px 20px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 16px;
  width: 100%;
}

button:hover:not(:disabled) {
  background: #1ed760;
}

button:disabled {
  background: #ccc;
  cursor: not-allowed;
}

.loading-spinner {
  text-align: center;
  padding: 20px;
}
```

## Module Configuration

Don't forget to import `HttpClientModule` in your `app.module.ts`:

```typescript
import { HttpClientModule } from '@angular/common/http';
import { FormsModule } from '@angular/forms';

@NgModule({
  imports: [
    // ... other imports
    HttpClientModule,
    FormsModule
  ],
  // ...
})
export class AppModule { }
```
