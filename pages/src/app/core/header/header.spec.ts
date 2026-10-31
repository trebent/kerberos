import { HarnessLoader } from '@angular/cdk/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatMenuHarness } from '@angular/material/menu/testing';
import { provideRouter } from '@angular/router';

import { ThemeService } from '../../services/theme.service';
import { expectNoAxeViolations } from '../../../testing/axe';
import { Header } from './header';

describe('Header', () => {
  let fixture: ComponentFixture<Header>;
  let loader: HarnessLoader;

  beforeEach(async () => {
    localStorage.clear();
    TestBed.configureTestingModule({ providers: [provideRouter([])] });

    fixture = TestBed.createComponent(Header);
    loader = TestbedHarnessEnvironment.documentRootLoader(fixture);
    await fixture.whenStable();
  });

  it('links to the API reference and the docs', async () => {
    const links = Array.from<HTMLAnchorElement>(
      fixture.nativeElement.querySelectorAll('nav a'),
    ).map((link) => [link.querySelector('.label')?.textContent, link.getAttribute('href')]);

    expect(links).toEqual([
      ['API reference', '/api'],
      ['Docs', 'https://github.com/trebent/kerberos/blob/main/docs/README.md'],
    ]);
    await expectNoAxeViolations(fixture);
  });

  it('switches the colour theme', async () => {
    const menu = await loader.getHarness(
      MatMenuHarness.with({ selector: '[aria-label^="Colour theme"]' }),
    );
    await menu.clickItem({ text: /Dark/ });

    expect(TestBed.inject(ThemeService).mode()).toBe('dark');
  });
});
