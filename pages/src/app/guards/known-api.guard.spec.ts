import { TestBed } from '@angular/core/testing';
import { PartialMatchRouteSnapshot, Route, UrlSegment } from '@angular/router';

import { knownApiGuard } from './known-api.guard';

describe('knownApiGuard', () => {
  const matches = (id: string) =>
    TestBed.runInInjectionContext(() =>
      knownApiGuard(
        {} as Route,
        [new UrlSegment('api', {}), new UrlSegment(id, {})],
        {} as PartialMatchRouteSnapshot,
      ),
    );

  it('matches the APIs in the catalogue', () => {
    expect(matches('gateway')).toBe(true);
    expect(matches('admin')).toBe(true);
    expect(matches('auth')).toBe(true);
  });

  it('rejects anything else', () => {
    expect(matches('nope')).toBe(false);
  });
});
