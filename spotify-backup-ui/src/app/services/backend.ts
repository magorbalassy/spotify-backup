import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface StatusResponse {
  hasToken: boolean;
  hasClientId: boolean;
  needsSetup: boolean;
  message: string;
}

export interface AuthSetupRequest {
  clientId: string;
  clientSecret: string;
}

export interface AuthSetupResponse {
  success: boolean;
  message: string;
  authUrl?: string;
}

export interface ErrorResponse {
  error: string;
}

@Injectable({
  providedIn: 'root',
})
export class Backend {
  private readonly apiUrl = '/api';

  constructor(private http: HttpClient) {}

  /**
   * Check the status of authentication (token availability and client credentials)
   */
  getStatus(): Observable<StatusResponse> {
    return this.http.get<StatusResponse>(`${this.apiUrl}/status`);
  }

  /**
   * Setup authentication by providing client ID and secret
   * Returns auth URL for Spotify authorization
   */
  setupAuth(clientId: string, clientSecret: string): Observable<AuthSetupResponse> {
    const request: AuthSetupRequest = {
      clientId,
      clientSecret
    };
    return this.http.post<AuthSetupResponse>(`${this.apiUrl}/auth/setup`, request);
  }

  /**
   * Start the OAuth flow (alternative to setupAuth if credentials already configured)
   */
  startAuth(): Observable<{ authUrl: string; message: string }> {
    return this.http.post<{ authUrl: string; message: string }>(
      `${this.apiUrl}/auth/start`,
      {}
    );
  }
}
