import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { expectNoAxeViolations } from '../testing/axe';
import { App } from './app';

describe('App', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
  });

  it('renders a skip link that moves focus to the main content', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const element: HTMLElement = fixture.nativeElement;

    element.querySelector<HTMLAnchorElement>('.skip-link')?.click();

    expect(document.activeElement).toBe(element.querySelector('main'));
  });

  it('has one landmark of each kind and no axe violations', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const element: HTMLElement = fixture.nativeElement;

    expect(element.querySelectorAll('main').length).toBe(1);
    expect(element.querySelectorAll('header').length).toBe(1);
    expect(element.querySelectorAll('footer').length).toBe(1);
    await expectNoAxeViolations(fixture);
  });
});
