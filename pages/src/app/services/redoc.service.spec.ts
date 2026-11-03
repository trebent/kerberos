import { provideHttpClient } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';

import { RedocService } from './redoc.service';

describe('RedocService', () => {
  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({ providers: [provideHttpClient()] });
  });
  afterEach(() => document.documentElement.classList.remove('light', 'dark'));

  it("sets Redoc's colour mode class and saved choice", () => {
    const redoc = TestBed.inject(RedocService);

    redoc.setColorMode('dark');
    expect(document.documentElement.classList).toContain('dark');
    expect(localStorage.getItem('redoc.colorMode')).toBe('dark');

    redoc.setColorMode('light');
    expect(document.documentElement.classList).toContain('light');
    expect(document.documentElement.classList).not.toContain('dark');
    expect(localStorage.getItem('redoc.colorMode')).toBe('light');
  });
});
