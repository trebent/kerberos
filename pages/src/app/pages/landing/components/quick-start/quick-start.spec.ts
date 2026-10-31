import { TestBed } from '@angular/core/testing';

import { expectNoAxeViolations } from '../../../../../testing/axe';
import { QuickStart } from './quick-start';

describe('QuickStart', () => {
  it('shows the commands with prompts and copies them without', async () => {
    const fixture = TestBed.createComponent(QuickStart);
    await fixture.whenStable();
    const element: HTMLElement = fixture.nativeElement;

    expect(element.querySelector('pre')?.textContent).toContain('$ krbctl compose -y --echo');
    expect(element.querySelectorAll('.prompt').length).toBe(4);
    await expectNoAxeViolations(fixture);
  });
});
