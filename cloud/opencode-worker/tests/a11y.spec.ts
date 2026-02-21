import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { execSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

test.describe('Accessibility checks', () => {
  test('should pass accessibility audit on local UI', async ({ page }) => {
    const url = execSync('alpine open test --print', { encoding: 'utf-8' }).trim();
    
    if (!url) {
      throw new Error('Failed to get URL from alpine open test --print');
    }

    await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    const reportPath = path.resolve(__dirname, '../../../artifacts/screenshots/a11y-report.json');
    fs.writeFileSync(reportPath, JSON.stringify(accessibilityScanResults, null, 2));
    console.log(`Accessibility report saved to: ${reportPath}`);

    expect(accessibilityScanResults.violations).toEqual([]);
  });
});
