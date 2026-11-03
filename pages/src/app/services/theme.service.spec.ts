import { TestBed } from '@angular/core/testing';

import { ThemeService } from './theme.service';

describe('ThemeService', () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => document.documentElement.style.removeProperty('color-scheme'));

  it('follows the system preference by default', () => {
    const theme = TestBed.inject(ThemeService);
    TestBed.tick();

    expect(theme.mode()).toBe('system');
    expect(document.documentElement.style.colorScheme).toBe('light dark');
  });

  it('applies and remembers an explicit choice', () => {
    const theme = TestBed.inject(ThemeService);
    theme.mode.set('dark');
    TestBed.tick();

    expect(document.documentElement.style.colorScheme).toBe('dark');
    expect(localStorage.getItem('kerberos.theme')).toBe('dark');
  });

  it('resolves the colour scheme, following the system preference in system mode', () => {
    const listeners: ((event: { matches: boolean }) => void)[] = [];
    vi.stubGlobal('matchMedia', () => ({
      matches: false,
      addEventListener: (_: string, listener: (event: { matches: boolean }) => void) =>
        listeners.push(listener),
    }));
    try {
      const theme = TestBed.inject(ThemeService);
      expect(theme.scheme()).toBe('light');

      listeners.forEach((listener) => listener({ matches: true }));
      expect(theme.scheme()).toBe('dark');

      theme.mode.set('light');
      expect(theme.scheme()).toBe('light');
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it('restores the saved choice', () => {
    localStorage.setItem('kerberos.theme', 'light');

    expect(TestBed.inject(ThemeService).mode()).toBe('light');
  });

  it('ignores unknown saved values', () => {
    localStorage.setItem('kerberos.theme', 'sepia');

    expect(TestBed.inject(ThemeService).mode()).toBe('system');
  });
});
