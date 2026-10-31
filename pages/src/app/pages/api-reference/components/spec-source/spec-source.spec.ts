import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { expectNoAxeViolations } from '../../../../../testing/axe';
import { APIS } from '../../../../utils/apis';
import { SpecSource } from './spec-source';

describe('SpecSource', () => {
  const setup = async () => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    const fixture = TestBed.createComponent(SpecSource);
    fixture.componentRef.setInput('api', APIS[0]);
    // Not whenStable(): the app is unstable until the spec request below is answered.
    TestBed.tick();
    return {
      fixture,
      http: TestBed.inject(HttpTestingController),
      element: fixture.nativeElement as HTMLElement,
    };
  };

  it('shows the YAML with highlighted keys', async () => {
    const { fixture, http, element } = await setup();

    http.expectOne('openapi/gateway.yaml').flush('# Gateway\nopenapi: "3.0.0"\npaths: {}\n');
    await fixture.whenStable();

    expect(element.querySelectorAll('.line').length).toBe(3);
    expect(element.querySelector('.tok-comment')?.textContent).toBe('# Gateway');
    expect(element.querySelector('.tok-key')?.textContent).toBe('openapi');
    expect(element.querySelector('.tok-string')?.textContent).toBe('"3.0.0"');
    expect(element.textContent).toContain('3 lines');
    await expectNoAxeViolations(fixture);
  });

  it('reports a failed download', async () => {
    const { fixture, http, element } = await setup();

    http.expectOne('openapi/gateway.yaml').flush('', { status: 404, statusText: 'Not Found' });
    await fixture.whenStable();

    expect(element.querySelector('[role="alert"]')?.textContent).toContain('could not be loaded');
  });
});
