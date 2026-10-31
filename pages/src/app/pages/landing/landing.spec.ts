import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { expectNoAxeViolations } from '../../../testing/axe';
import { Landing } from './landing';

describe('Landing', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
  });

  it('links to each API reference and to the docs', async () => {
    const fixture = TestBed.createComponent(Landing);
    await fixture.whenStable();
    const element: HTMLElement = fixture.nativeElement;

    const hrefs = Array.from(element.querySelectorAll('a')).map((a) => a.getAttribute('href'));
    expect(hrefs).toEqual(
      expect.arrayContaining([
        '/api',
        '/api/gateway',
        '/api/admin',
        '/api/auth',
        'https://github.com/trebent/kerberos/blob/main/docs/README.md',
      ]),
    );
    expect(element.querySelectorAll('h1').length).toBe(1);
    await expectNoAxeViolations(fixture);
  });
});
