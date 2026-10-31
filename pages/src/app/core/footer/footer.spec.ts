import { TestBed } from '@angular/core/testing';

import { expectNoAxeViolations } from '../../../testing/axe';
import { Footer } from './footer';

describe('Footer', () => {
  it('renders accessible link lists', async () => {
    const fixture = TestBed.createComponent(Footer);
    await fixture.whenStable();

    const element: HTMLElement = fixture.nativeElement;
    expect(element.querySelectorAll('nav').length).toBe(2);
    expect(element.textContent).toContain(String(new Date().getFullYear()));
    await expectNoAxeViolations(fixture);
  });
});
