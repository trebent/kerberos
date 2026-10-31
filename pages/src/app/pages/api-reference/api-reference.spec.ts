import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router, withComponentInputBinding } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';

import { RedocService } from '../../services/redoc.service';
import { expectNoAxeViolations } from '../../../testing/axe';
import { routes } from '../../app.routes';
import { ApiReference } from './api-reference';

describe('ApiReference', () => {
  let render: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    render = vi.fn(async () => undefined);
    TestBed.configureTestingModule({
      providers: [
        provideRouter(routes, withComponentInputBinding()),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: RedocService, useValue: { render } },
      ],
    });
  });

  it('renders the selected API with Redoc', async () => {
    const harness = await RouterTestingHarness.create();
    const page = await harness.navigateByUrl('/api/admin', ApiReference);
    await harness.fixture.whenStable();
    const element: HTMLElement = harness.routeNativeElement!;

    expect(page).toBeInstanceOf(ApiReference);
    expect(element.querySelector('h1')?.textContent).toBe('Admin API');
    expect(element.querySelector('[aria-selected="true"]')?.textContent?.trim()).toBe('Admin');
    expect(render).toHaveBeenCalledWith(
      'openapi/admin.yaml',
      expect.any(HTMLElement),
      expect.anything(),
    );
    expect(document.title).toBe('Admin API · Kerberos');
    await expectNoAxeViolations(harness.fixture);
  });

  it('switches to the source view through the query string', async () => {
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/api/auth', ApiReference);
    const toggles = harness.routeNativeElement!.querySelectorAll<HTMLButtonElement>(
      'mat-button-toggle button',
    );

    // Not MatButtonToggleHarness: it waits for stability, which the spec request below blocks.
    toggles[1].click();
    await vi.waitFor(() => {
      TestBed.tick();
      TestBed.inject(HttpTestingController).expectOne('openapi/auth_basic.yaml').flush('a: 1\n');
    });
    await harness.fixture.whenStable();

    expect(TestBed.inject(Router).url).toBe('/api/auth?view=source');
    expect(harness.routeNativeElement?.querySelector('app-spec-source .line')).not.toBeNull();
  });

  it('redirects /api to the gateway API and old api.html links to the new routes', async () => {
    const harness = await RouterTestingHarness.create();

    await harness.navigateByUrl('/api');
    expect(harness.routeNativeElement?.querySelector('h1')?.textContent).toBe('Gateway API');

    const navigation = harness.navigateByUrl('/api.html?api=auth&view=source');
    await vi.waitFor(() =>
      TestBed.inject(HttpTestingController).expectOne('openapi/auth_basic.yaml').flush('a: 1\n'),
    );
    await navigation;
    expect(harness.routeNativeElement?.querySelector('h1')?.textContent).toBe('Basic-auth API');
    expect(harness.routeNativeElement?.querySelector('app-spec-source')).not.toBeNull();
  });
});
