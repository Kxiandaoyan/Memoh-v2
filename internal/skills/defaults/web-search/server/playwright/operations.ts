/**
 * Browser Operations - Common browser operations using Playwright Page API
 */

import { Page } from 'playwright-core';

export interface NavigateOptions {
  url: string;
  waitUntil?: 'load' | 'domcontentloaded' | 'networkidle';
  timeout?: number;
}

export interface ScreenshotOptions {
  format?: 'png' | 'jpeg';
  fullPage?: boolean;
  quality?: number;
}

export interface EvaluateOptions {
  expression: string;
  args?: any[];
}

/**
 * Navigate to URL
 */
export async function navigate(page: Page, options: NavigateOptions): Promise<void> {
  console.log(`[Operations] Navigating to: ${options.url}`);

  await page.goto(options.url, {
    waitUntil: options.waitUntil || 'domcontentloaded',
    timeout: options.timeout || 30000
  });

  console.log(`[Operations] Navigation complete: ${page.url()}`);
}

/**
 * Take screenshot
 */
export async function screenshot(page: Page, options: ScreenshotOptions = {}): Promise<Buffer> {
  console.log(`[Operations] Taking screenshot (format: ${options.format || 'png'})`);

  const buffer = await page.screenshot({
    type: options.format || 'png',
    fullPage: options.fullPage || false,
    quality: options.quality
  });

  console.log(`[Operations] Screenshot captured (${buffer.length} bytes)`);
  return buffer;
}

/**
 * Get page content (HTML)
 */
export async function getContent(page: Page): Promise<string> {
  console.log(`[Operations] Getting page content`);
  const content = await page.content();
  console.log(`[Operations] Content retrieved (${content.length} chars)`);
  return content;
}

/**
 * Get page text content
 */
export async function getTextContent(page: Page): Promise<string> {
  console.log(`[Operations] Getting text content`);

  const text = await page.textContent('body') || '';

  console.log(`[Operations] Text content retrieved (${text.length} chars)`);
  return text;
}

/**
 * Evaluate JavaScript expression
 */
export async function evaluate(page: Page, options: EvaluateOptions): Promise<any> {
  console.log(`[Operations] Evaluating expression`);

  const result = await page.evaluate((args: any) => {
    // eslint-disable-next-line no-eval
    return eval(args.expression);
  }, { expression: options.expression, args: options.args || [] });

  console.log(`[Operations] Evaluation complete`);
  return result;
}

/**
 * Wait for selector
 */
export async function waitForSelector(
  page: Page,
  selector: string,
  timeout: number = 10000
): Promise<void> {
  console.log(`[Operations] Waiting for selector: ${selector}`);

  await page.waitForSelector(selector, { timeout });

  console.log(`[Operations] Selector found: ${selector}`);
}

/**
 * Click element
 */
export async function click(page: Page, selector: string): Promise<void> {
  console.log(`[Operations] Clicking element: ${selector}`);

  await page.click(selector);

  console.log(`[Operations] Click complete: ${selector}`);
}

/**
 * Type text into input
 */
export async function type(page: Page, selector: string, text: string): Promise<void> {
  console.log(`[Operations] Typing into: ${selector}`);

  await page.fill(selector, text);

  console.log(`[Operations] Type complete: ${selector}`);
}

/**
 * Get current URL
 */
export function getCurrentUrl(page: Page): string {
  return page.url();
}

/**
 * Get page title
 */
export async function getTitle(page: Page): Promise<string> {
  return await page.title();
}

/**
 * Go back in history
 */
export async function goBack(page: Page): Promise<void> {
  console.log(`[Operations] Going back`);
  await page.goBack();
}

/**
 * Go forward in history
 */
export async function goForward(page: Page): Promise<void> {
  console.log(`[Operations] Going forward`);
  await page.goForward();
}

/**
 * Reload page
 */
export async function reload(page: Page): Promise<void> {
  console.log(`[Operations] Reloading page`);
  await page.reload();
}

/**
 * Wait for navigation
 */
export async function waitForNavigation(
  page: Page,
  timeout: number = 30000
): Promise<void> {
  console.log(`[Operations] Waiting for navigation`);
  await page.waitForNavigation({ timeout });
}

/**
 * Set viewport size
 */
export async function setViewport(page: Page, width: number, height: number): Promise<void> {
  console.log(`[Operations] Setting viewport: ${width}x${height}`);
  await page.setViewportSize({ width, height });
}
