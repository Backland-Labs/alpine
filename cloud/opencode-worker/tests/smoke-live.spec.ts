import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';
import * as path from 'path';

test.describe('Live smoke test', () => {
  test('should load OpenCode UI from deployed Worker', async ({ page }) => {
    const url = execSync('alpine open test --print', { encoding: 'utf-8' }).trim();
    
    if (!url) {
      throw new Error('Failed to get URL from alpine open test --print');
    }

    console.log(`Visiting live URL: ${url}`);
    
    await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });

    await page.waitForSelector('[data-testid="opencode-ui"], .opencode-container, #opencode-root', {
      timeout: 15000,
    }).catch(async () => {
      const bodyText = await page.textContent('body').catch(() => '');
      if (bodyText?.includes('OpenCode') || bodyText?.includes('sandbox')) {
        console.log('Found OpenCode/sandbox reference in page body');
        return;
      }
      throw new Error('OpenCode UI marker not found');
    });

    const screenshotPath = path.resolve(__dirname, '../../../artifacts/screenshots/live-ui.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });
    console.log(`Screenshot saved to: ${screenshotPath}`);

    await expect(page).toHaveTitle(/OpenCode|Sandbox/i);
  });
});
