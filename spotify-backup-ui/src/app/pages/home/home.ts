import { Component, OnInit, signal } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Backend, StatusResponse } from '../../services/backend';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-home',
  imports: [
    CommonModule,
    MatCardModule, 
    MatButtonModule, 
    MatIconModule,
    MatProgressSpinnerModule
  ],
  templateUrl: './home.html',
  styleUrl: './home.css',
})
export class Home implements OnInit {
  status = signal<StatusResponse | null>(null);
  loading = signal(false);
  error = signal<string | null>(null);

  constructor(private backend: Backend) {}

  ngOnInit() {
    this.checkStatus();
  }

  checkStatus() {
    this.loading.set(true);
    this.error.set(null);
    
    this.backend.getStatus().subscribe({
      next: (status) => {
        this.status.set(status);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Failed to check status:', err);
        this.error.set('Failed to connect to backend. Please ensure the Go server is running and accessible.');
        this.loading.set(false);
      }
    });
  }
}
