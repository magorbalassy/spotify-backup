import { Routes } from '@angular/router';
import { Home } from './pages/home/home';
import { Config } from './pages/config/config';

export const routes: Routes = [
  { path: '', redirectTo: '/home', pathMatch: 'full' },
  { path: 'home', component: Home },
  { path: 'config', component: Config },
  { path: '**', redirectTo: '/home' }
];
