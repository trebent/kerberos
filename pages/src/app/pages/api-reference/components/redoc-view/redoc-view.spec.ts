import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { RedocService } from '../../../../services/redoc.service';
import { expectNoAxeViolations } from '../../../../../testing/axe';
import { APIS } from '../../../../utils/apis';
import { RedocView } from './redoc-view';

describe('RedocView', () => {
  const setup = async (render: RedocService['render']) => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), { provide: RedocService, useValue: { render } }],
    });
    const fixture = TestBed.createComponent(RedocView);
    fixture.componentRef.setInput('api', APIS[1]);
    await fixture.whenStable();
    return { fixture, element: fixture.nativeElement as HTMLElement };
  };

  it('renders the spec with Redoc', async () => {
    const render = vi.fn(async (_url: string, host: HTMLElement) => {
      host.innerHTML = '<p>rendered</p>';
    });
    const { fixture, element } = await setup(render);
    await fixture.whenStable();

    expect(render).toHaveBeenCalledWith(
      'openapi/admin.yaml',
      expect.any(HTMLElement),
      expect.anything(),
    );
    expect(element.querySelector('[role="status"]')).toBeNull();
    expect(element.querySelector('.redoc')?.textContent).toContain('rendered');
    await expectNoAxeViolations(fixture);
  });

  it('points to the source view when Redoc fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const { fixture, element } = await setup(() => Promise.reject(new Error('offline')));
    await fixture.whenStable();

    expect(element.querySelector('[role="alert"]')?.textContent).toContain('could not be loaded');
    expect(element.querySelector('[role="alert"] a')?.getAttribute('href')).toContain(
      'view=source',
    );
    await expectNoAxeViolations(fixture);
  });
});
