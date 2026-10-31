import { ComponentFixture } from '@angular/core/testing';
import axe from 'axe-core';

/**
 * Runs axe-core against a rendered component and fails with a readable list of violations.
 * jsdom has no layout engine, so color contrast is checked in a real browser instead.
 */
export async function expectNoAxeViolations(fixture: ComponentFixture<unknown>): Promise<void> {
  await fixture.whenStable();

  const { violations } = await axe.run(fixture.nativeElement as HTMLElement, {
    rules: { 'color-contrast': { enabled: false } },
  });

  const report = violations.map(
    (violation) =>
      `${violation.id}: ${violation.help}\n` +
      violation.nodes.map((node) => `  ${node.html}`).join('\n'),
  );
  expect(report).toEqual([]);
}
