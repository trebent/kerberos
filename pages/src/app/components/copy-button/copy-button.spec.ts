import { Clipboard } from '@angular/cdk/clipboard';
import { TestBed } from '@angular/core/testing';

import { expectNoAxeViolations } from '../../../testing/axe';
import { CopyButton } from './copy-button';

describe('CopyButton', () => {
  const setup = async (copied: boolean) => {
    const copy = vi.fn(() => copied);
    TestBed.configureTestingModule({ providers: [{ provide: Clipboard, useValue: { copy } }] });
    const fixture = TestBed.createComponent(CopyButton);
    fixture.componentRef.setInput('text', 'make run');
    fixture.componentRef.setInput('subject', 'the commands');
    await fixture.whenStable();
    return { fixture, copy, element: fixture.nativeElement as HTMLElement };
  };

  it('copies the text and confirms it', async () => {
    const { fixture, copy, element } = await setup(true);

    element.querySelector('button')?.click();
    await fixture.whenStable();

    expect(copy).toHaveBeenCalledWith('make run');
    expect(element.querySelector('button')?.textContent).toContain('Copy the commands');
    expect(element.querySelector('[role="status"]')?.textContent).toContain('Copied');
    await expectNoAxeViolations(fixture);
  });

  it('reports a failure', async () => {
    const { fixture, element } = await setup(false);

    element.querySelector('button')?.click();
    await fixture.whenStable();

    expect(element.querySelector('[role="status"]')?.textContent).toContain('Copying failed');
  });
});
