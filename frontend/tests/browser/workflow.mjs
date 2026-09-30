import assert from 'node:assert/strict';
import { execFileSync, spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { after, before, test as nodeTest } from 'node:test';
import { setTimeout as delay } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const repo = fileURLToPath(new URL('../../../', import.meta.url));
const original = 'ref: sun\nref: hiking\nTowel';
const proposal = 'ref: cold\nref: hiking';
const legacy = { selection: original, done: ['Sunscreen'], doneDisplay: 'hidden' };
const test = (name, run) => nodeTest(name, { timeout: 30000 }, run);
const artifacts = process.env.PACKING_TEST_ARTIFACTS;
let temporary;
let server;
let browser;
let origin;

before(async () => {
  temporary = await mkdtemp(join(tmpdir(), 'packing-browser-'));
  if (artifacts) await mkdir(artifacts, { recursive: true });
  const groups = join(temporary, 'groups');
  await mkdir(groups);
  await Promise.all([
    writeFile(join(groups, 'sun'), 'Sunscreen\nSunglasses'),
    writeFile(join(groups, 'hiking'), 'Sunscreen\nBoots'),
    writeFile(join(groups, 'cold'), 'Sunscreen\nGloves'),
  ]);
  // Build the embedded UI and isolate the server binary and group fixtures.
  execFileSync('make', ['frontend'], { cwd: repo, stdio: 'inherit' });
  const binary = join(temporary, 'packing-server');
  execFileSync('go', ['build', '-o', binary, './cmd/packing-server'], { cwd: repo, stdio: 'inherit' });

  const portProbe = createServer();
  portProbe.listen(0, '127.0.0.1');
  await once(portProbe, 'listening');
  const { port } = portProbe.address();
  await new Promise(resolve => portProbe.close(resolve));
  origin = `http://127.0.0.1:${port}`;
  server = spawn(binary, ['serve', '--groups-dir', groups, '--port', String(port)], { stdio: ['ignore', 'ignore', 'pipe'] });
  let serverError = '';
  server.stderr.on('data', chunk => { serverError += chunk; });
  for (let attempt = 0; attempt < 100; attempt++) {
    if (server.exitCode !== null) throw new Error(`Server exited: ${serverError}`);
    try {
      const response = await fetch(origin);
      if (response.ok) break;
    } catch {
      // Connection refusal is expected while the isolated server starts.
    }
    if (attempt === 99) throw new Error(`Server did not start: ${serverError}`);
    await delay(50);
  }
  browser = await chromium.launch();
}, { timeout: 120000 });

after(async () => {
  try {
    await browser?.close();
  } finally {
    if (server && server.exitCode === null) {
      const exited = once(server, 'exit');
      server.kill('SIGTERM');
      await exited;
    }
    if (temporary) await rm(temporary, { recursive: true, force: true });
  }
});

async function openPage(t, saved = legacy, setup = async () => {}) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  t.after(() => context.close());
  await context.addInitScript(saved => {
    if (!sessionStorage.getItem('fixture-initialized')) {
      localStorage.setItem('appState', JSON.stringify(saved));
      sessionStorage.setItem('fixture-initialized', 'yes');
    }
  }, saved);
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  t.after(() => assert.deepEqual(errors, [], 'No browser runtime errors'));
  await setup(page);
  await page.goto(origin);
  return page;
}

const button = (page, name) => page.getByRole('button', { name, exact: true });
const editor = page => page.getByRole('textbox', { name: 'Draft selection' });
const ready = page => page.getByText('Preview ready.', { exact: true }).waitFor();
const savedState = page => page.evaluate(() => JSON.parse(localStorage.getItem('appState')));

async function edit(page, selection) {
  await editor(page).fill(selection);
  await page.getByText('Updating preview…', { exact: true }).waitFor();
  await ready(page);
}

async function isCurrent(page) {
  await page.waitForFunction(() => [...document.querySelectorAll('button')].some(b => b.textContent === 'Current list' && b.getAttribute('aria-pressed') === 'true'));
}

test('review a group swap, preserve shared items and packing progress, then apply the exact preview', async t => {
  const page = await openPage(t);
  await ready(page);
  await isCurrent(page);
  const baseline = (await savedState(page)).committed;
  await edit(page, proposal);
  await page.getByText('1 item added · 2 removed · 2 retained', { exact: true }).waitFor();
  assert.deepEqual((await savedState(page)).committed, baseline);
  await page.getByText('Source changes (1) — items still included', { exact: true }).click();
  await page.getByText('After: cold, hiking', { exact: true }).waitFor();
  if (artifacts) await page.screenshot({ path: join(artifacts, 'selection-changes-desktop.png'), fullPage: true });

  await button(page, 'Proposed list').click();
  await page.getByText('Gloves', { exact: true }).waitFor();
  await page.getByText('Sunscreen', { exact: true }).first().click();
  assert.deepEqual((await savedState(page)).done, ['Sunscreen']);
  assert.equal(await button(page, 'Clear packing progress').count(), 0);
  await button(page, 'graph view').click();
  await page.locator('svg').waitFor();
  await button(page, 'Current list').click();
  await button(page, 'list view').click();
  await page.getByText('Towel', { exact: true }).waitFor();
  assert.equal(await page.getByText('Gloves', { exact: true }).count(), 0);

  let requestsAfterApply = 0;
  page.on('request', request => { if (request.url().endsWith('/selection/')) requestsAfterApply++; });
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  await page.waitForFunction(text => JSON.parse(localStorage.getItem('appState')).committed.selection === text, proposal);
  const accepted = await savedState(page);
  assert.equal(accepted.committed.selection, proposal);
  assert.deepEqual(accepted.done, ['Sunscreen']);
  assert.equal(requestsAfterApply, 0, 'Apply must accept the preview without resolving again');
  await page.reload();
  await ready(page);
  assert.deepEqual((await savedState(page)).committed, accepted.committed);
});

test('reload restores an invalid draft; Discard restores the accepted snapshot without a request', async t => {
  const page = await openPage(t);
  await ready(page);
  const baseline = (await savedState(page)).committed;
  await editor(page).fill('ref:');
  await page.getByText(/Preview failed:/).waitFor();
  assert.equal(await button(page, 'Apply changes').isDisabled(), true);
  await page.reload();
  await page.getByText('Unapplied draft restored.', { exact: true }).waitFor();
  await page.getByText(/Preview failed:/).waitFor();
  assert.equal(await editor(page).inputValue(), 'ref:');
  assert.deepEqual((await savedState(page)).committed, baseline);
  let requests = 0;
  page.on('request', request => { if (request.url().endsWith('/selection/')) requests++; });
  await button(page, 'Discard changes').click();
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), original);
  assert.deepEqual((await savedState(page)).committed, baseline);
  assert.equal(requests, 0);
});

test('Discard rejects a late resolution and a late Simplify response', async t => {
  const page = await openPage(t);
  await ready(page);
  let releaseResolution;
  let resolutionStarted;
  const started = new Promise(resolve => { resolutionStarted = resolve; });
  const gate = new Promise(resolve => { releaseResolution = resolve; });
  await page.route('**/selection/', async route => {
    if (route.request().postData() === proposal) {
      resolutionStarted();
      await gate;
    }
    await route.continue();
  });
  await editor(page).fill(proposal);
  await started;
  const finished = page.waitForResponse(response => response.url().endsWith('/selection/'));
  await button(page, 'Discard changes').click();
  releaseResolution();
  await finished;
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), original);

  let releaseSimplify;
  let simplifyStarted;
  const simplifying = new Promise(resolve => { simplifyStarted = resolve; });
  const simplifyGate = new Promise(resolve => { releaseSimplify = resolve; });
  await page.route('**/simplify/', async route => {
    simplifyStarted();
    await simplifyGate;
    await route.fulfill({ json: { selection: '', removedRefs: ['sun', 'hiking'], removedItems: ['Towel'] } });
  });
  await button(page, 'Simplify').click();
  await simplifying;
  const simplified = page.waitForResponse(response => response.url().endsWith('/simplify/'));
  await button(page, 'Discard changes').click();
  releaseSimplify();
  await simplified;
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), original);
  assert.equal((await savedState(page)).committed.selection, original);
});

test('Simplify is reviewable, and drafts with no item changes can be accepted', async t => {
  const selection = 'ref: sun\nSunscreen\n# keep this comment';
  const page = await openPage(t, { selection, done: [] });
  await ready(page);
  await button(page, 'Simplify').click();
  await page.waitForFunction(() => document.querySelector('textarea').value === 'ref: sun\n# keep this comment');
  await ready(page);
  await page.getByText('Selection changed; no change to overall items.', { exact: true }).waitFor();
  assert.equal((await savedState(page)).committed.selection, selection);
  assert.equal(await button(page, 'Apply changes').isEnabled(), true);
  await button(page, 'Discard changes').click();
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), selection);
  await edit(page, `${selection}\n# reviewed`);
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  assert.equal((await savedState(page)).committed.selection, `${selection}\n# reviewed`);
});

test('an empty selection removes all items and can be applied', async t => {
  const page = await openPage(t);
  await ready(page);
  await edit(page, '');
  await page.getByText('0 items added · 4 removed · 0 retained', { exact: true }).waitFor();
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  await page.getByText('This selection contains no items or groups.', { exact: true }).waitFor();
  assert.deepEqual((await savedState(page)).committed, { selection: '', groups: [] });
  assert.deepEqual((await savedState(page)).done, ['Sunscreen']);
});

test('group-file drift remains a proposed change on reload', async t => {
  const committed = { selection: 'ref: sun', groups: [{ name: 'sun', contents: { refs: [], items: ['Old hat'] } }] };
  const page = await openPage(t, { version: 2, committed, draftText: committed.selection, done: [] });
  await ready(page);
  await page.getByText(/Referenced groups have changed since/).waitFor();
  assert.deepEqual((await savedState(page)).committed, committed);
  await button(page, 'Discard changes').click();
  await isCurrent(page);
  await page.getByText('Old hat', { exact: true }).waitFor();
});

test('storage write errors are visible while the accepted in-memory selection remains usable', async t => {
  const page = await openPage(t);
  await ready(page);
  await page.evaluate(() => {
    Storage.prototype.setItem = function () { throw new DOMException('Test quota exceeded', 'QuotaExceededError'); };
  });
  await edit(page, 'Passport');
  await page.getByText(/Could not save on this device:/).waitFor();
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  await page.getByText('Passport', { exact: true }).waitFor();
  await page.getByText(/Your changes remain in this tab/).waitFor();
});

test('mobile layout stacks the editor and review without horizontal scrolling', async t => {
  const page = await openPage(t);
  await page.setViewportSize({ width: 390, height: 844 });
  await ready(page);
  await edit(page, proposal);
  const layout = await page.evaluate(() => {
    const panels = [...document.querySelectorAll('.packing-panel')].map(panel => panel.getBoundingClientRect());
    return { stacked: panels[1].top >= panels[0].bottom, fits: document.documentElement.scrollWidth <= innerWidth };
  });
  assert.deepEqual(layout, { stacked: true, fits: true });
  if (artifacts) await page.screenshot({ path: join(artifacts, 'selection-changes-mobile.png'), fullPage: true });
});

test('the selection editor fills the available pane height before scrolling internally', async t => {
  const page = await openPage(t);
  await ready(page);
  await edit(page, Array.from({ length: 90 }, (_, i) => `Item ${i}`).join('\n'));
  for (const height of [1200, 720]) {
    await page.setViewportSize({ width: 1280, height });
    const layout = await page.evaluate(() => {
      const input = document.querySelector('.selection-editor');
      const bounds = input.getBoundingClientRect();
      const panel = document.querySelector('.selection-panel').getBoundingClientRect();
      return {
        fillsPane: Math.abs(bounds.bottom - panel.bottom) < 2,
        fitsWindow: bounds.bottom <= innerHeight,
        scrolls: input.scrollHeight > input.clientHeight,
        height: bounds.height,
      };
    });
    assert.equal(layout.fillsPane, true, `Editor must use remaining pane height at ${height}px`);
    assert.equal(layout.fitsWindow, true);
    assert.equal(layout.scrolls, true, 'Long content scrolls after the editor fills the pane');
    if (height === 1200) assert.ok(layout.height > 600, 'Tall windows provide more than the old 30 rows');
  }
});
