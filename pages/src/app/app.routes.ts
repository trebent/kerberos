import { inject } from '@angular/core';
import { ResolveFn, Router, Routes } from '@angular/router';

import { knownApiGuard } from './guards/known-api.guard';
import { findApi } from './utils/apis';

const SITE = 'Kerberos';

const apiTitle: ResolveFn<string> = (route) =>
  `${findApi(route.paramMap.get('api'))?.name ?? 'API reference'} · ${SITE}`;

export const routes: Routes = [
  {
    path: '',
    title: 'Kerberos API gateway',
    loadComponent: () => import('./pages/landing/landing').then((m) => m.Landing),
  },
  { path: 'api', pathMatch: 'full', redirectTo: 'api/gateway' },
  {
    path: 'api/:api',
    title: apiTitle,
    canMatch: [knownApiGuard],
    loadComponent: () => import('./pages/api-reference/api-reference').then((m) => m.ApiReference),
  },
  {
    // The old static site served the reference at api.html?api=<id>&view=<view>.
    path: 'api.html',
    redirectTo: ({ queryParams }) => {
      const view = queryParams['view'] === 'source' ? { view: 'source' } : {};
      return inject(Router).createUrlTree(['/api', findApi(queryParams['api'])?.id ?? 'gateway'], {
        queryParams: view,
      });
    },
  },
  { path: '**', redirectTo: '' },
];
