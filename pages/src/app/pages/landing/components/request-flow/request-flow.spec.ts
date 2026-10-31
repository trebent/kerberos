import { TestBed } from '@angular/core/testing';

import { expectNoAxeViolations } from '../../../../../testing/axe';
import { RequestFlow } from './request-flow';

describe('RequestFlow', () => {
  it('lists the pipeline stages in order', async () => {
    const fixture = TestBed.createComponent(RequestFlow);
    await fixture.whenStable();
    const element: HTMLElement = fixture.nativeElement;

    const stages = Array.from(element.querySelectorAll('ol > li h3')).map((h) =>
      h.textContent?.trim(),
    );
    expect(stages).toEqual(['Observe', 'Route', 'Authenticate', 'Validate', 'Forward']);
    await expectNoAxeViolations(fixture);
  });
});
