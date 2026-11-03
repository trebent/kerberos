import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { RedocService } from '../../../../services/redoc.service';
import { ThemeService } from '../../../../services/theme.service';
import { expectNoAxeViolations } from '../../../../../testing/axe';
import { APIS } from '../../../../utils/apis';
import { RedocView } from './redoc-view';

describe('RedocView', () => {
  // ThemeService restores a saved theme; start each test from the default.
  beforeEach(() => localStorage.clear());

  const setup = async (render: RedocService['render']) => {
    const setColorMode = vi.fn();
    TestBed.configureTestingModule({
      providers: [provideRouter([]), { provide: RedocService, useValue: { render, setColorMode } }],
    });
    const fixture = TestBed.createComponent(RedocView);
    fixture.componentRef.setInput('api', APIS[1]);
    await fixture.whenStable();
    return { fixture, setColorMode, element: fixture.nativeElement as HTMLElement };
  };

  it('renders the spec with Redoc', async () => {
    const render = vi.fn(async (_url: string, element: HTMLElement) => {
      element.innerHTML = '<p>rendered</p>';
      return { destroy: vi.fn() };
    });
    const { fixture, element } = await setup(render);
    await fixture.whenStable();

    expect(render).toHaveBeenCalledWith(
      'openapi/admin.yaml',
      expect.any(HTMLElement),
      expect.objectContaining({ hideDownloadButtons: true }),
    );
    expect(element.querySelector('[role="status"]')).toBeNull();
    expect(element.querySelector('.redoc')?.textContent).toContain('rendered');
    await expectNoAxeViolations(fixture);
  });

  it('unmounts the previous reference when the API changes and when destroyed', async () => {
    const handles = [{ destroy: vi.fn() }, { destroy: vi.fn() }];
    const render = vi.fn(async () => handles[render.mock.calls.length - 1]);
    const { fixture } = await setup(render);
    await fixture.whenStable();

    fixture.componentRef.setInput('api', APIS[2]);
    await fixture.whenStable();
    expect(render).toHaveBeenLastCalledWith(
      'openapi/auth_basic.yaml',
      expect.any(HTMLElement),
      expect.anything(),
    );
    expect(handles[0].destroy).toHaveBeenCalled();
    expect(handles[1].destroy).not.toHaveBeenCalled();

    fixture.destroy();
    expect(handles[1].destroy).toHaveBeenCalled();
  });

  it("follows the site's colour scheme", async () => {
    const { fixture, setColorMode } = await setup(async () => ({ destroy: vi.fn() }));
    expect(setColorMode).toHaveBeenLastCalledWith('light');

    TestBed.inject(ThemeService).mode.set('dark');
    await fixture.whenStable();
    expect(setColorMode).toHaveBeenLastCalledWith('dark');
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
