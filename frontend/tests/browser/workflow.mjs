import assert from 'node:assert/strict';
import { execFileSync, spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
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
let savedServer;
let savedOrigin;
let selectionFile;

before(async () => {
  temporary = await mkdtemp(join(tmpdir(), 'packing-browser-'));
  if (artifacts) await mkdir(artifacts, { recursive: true });
  const groups = join(temporary, 'groups');
  await mkdir(groups);
  await Promise.all([
    writeFile(join(groups, 'sun'), 'Sunscreen\nSunglasses'),
    writeFile(join(groups, 'hiking'), 'Sunscreen\nBoots'),
    writeFile(join(groups, 'cold'), 'Sunscreen\nGloves'),
    writeFile(join(groups, 'group_a'), 'ref: group_b\na_2\na_1'),
    writeFile(join(groups, 'group_b'), 'b_2\nb_1'),
    writeFile(join(groups, 'bundle'), 'ref: group_a\nref: branch\nref: passthrough'),
    writeFile(join(groups, 'branch'), 'ref: group_b\nb_1\nc_1'),
    writeFile(join(groups, 'passthrough'), 'ref: singleton'),
    writeFile(join(groups, 'singleton'), 's_1'),
    writeFile(join(groups, 'empty'), ''),
  ]);
  // Build the embedded UI and isolate the server binary and group fixtures.
  execFileSync('make', ['frontend'], { cwd: repo, stdio: 'inherit' });
  const binary = join(temporary, 'packing-server');
  execFileSync('go', ['build', '-o', binary, './cmd/packing-server'], { cwd: repo, stdio: 'inherit' });

  ({ server, origin } = await startServer(binary, groups));
  selectionFile = join(temporary, 'trip.selection');
  ({ server: savedServer, origin: savedOrigin } = await startServer(binary, groups, ['--selection-file', selectionFile]));
  browser = await chromium.launch();
}, { timeout: 120000 });

async function startServer(binary, groups, extraArgs = []) {
  const portProbe = createServer();
  portProbe.listen(0, '127.0.0.1');
  await once(portProbe, 'listening');
  const { port } = portProbe.address();
  await new Promise(resolve => portProbe.close(resolve));
  const origin = `http://127.0.0.1:${port}`;
  const server = spawn(binary, ['serve', '--groups-dir', groups, '--port', String(port), ...extraArgs], { stdio: ['ignore', 'ignore', 'pipe'] });
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
  return { server, origin };
}

after(async () => {
  try {
    await browser?.close();
  } finally {
    for (const process of [server, savedServer]) {
      if (process && process.exitCode === null) {
        const exited = once(process, 'exit');
        process.kill('SIGTERM');
        await exited;
      }
    }
    if (temporary) await rm(temporary, { recursive: true, force: true });
  }
});

async function openPage(t, saved = legacy, setup = async () => {}, pageOrigin = origin) {
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
  await page.goto(pageOrigin);
  return page;
}

const button = (page, name) => page.getByRole('button', { name, exact: true });
const editor = page => page.getByRole('textbox', { name: 'Draft selection' });
const ready = page => page.getByText('Preview ready.', { exact: true }).waitFor();
const savedState = page => page.evaluate(() => JSON.parse(localStorage.getItem(selectionStorageKey || 'appState')));
const groupItems = page => page.locator('[aria-label="Group items"]');
const flatItems = page => groupItems(page).locator('li').allTextContents();

async function assertFlatItems(page, expected) {
  await groupItems(page).waitFor();
  // Elm renders on the next animation frame; wait for the requested list rather
  // than reading the previous DOM immediately after a click.
  await page.waitForFunction(expected => {
    const items = [...document.querySelectorAll('[aria-label="Group items"] li')].map(item => item.textContent);
    return JSON.stringify(items) === JSON.stringify(expected);
  }, expected);
  assert.deepEqual(await flatItems(page), expected);
}

async function edit(page, selection) {
  await editor(page).fill(selection);
  await page.getByText('Updating preview…', { exact: true }).waitFor();
  await ready(page);
}

async function isCurrent(page) {
  await page.waitForFunction(() => [...document.querySelectorAll('button')].some(b => b.textContent === 'Current list' && b.getAttribute('aria-pressed') === 'true'));
}

test('group headings open a flat descendant list and return without fetching or changing the selection', async t => {
  const page = await openPage(t, { selection: 'ref: group_a\nref: cold', done: [], doneDisplay: 'marked' });
  await ready(page);
  await isCurrent(page);
  await button(page, 'show group links').click();
  const baseline = await savedState(page);
  const requests = [];
  page.on('request', request => { requests.push(request.url()); });

  await button(page, 'group_a').focus();
  await page.keyboard.press('Enter');
  await assertFlatItems(page, ['a_1', 'a_2', 'b_1', 'b_2']);
  assert.deepEqual(await groupItems(page).locator('h3').allTextContents(), ['group_a']);
  assert.equal(await groupItems(page).locator('h4, a').count(), 0);
  assert.equal(await button(page, 'show group links').count(), 0);
  assert.equal(await page.getByText('Gloves', { exact: true }).count(), 0);

  await button(page, 'Back to list').click();
  await page.locator('#group_a').getByRole('link', { name: 'group_b', exact: true }).waitFor();
  await button(page, 'group_b').click();
  await assertFlatItems(page, ['b_1', 'b_2']);
  assert.deepEqual(await savedState(page), baseline);
  assert.deepEqual(requests, [], 'Group navigation uses the existing snapshot');
});

test('container lists include deep, shared, and collapsed items once and explain empty groups', async t => {
  const page = await openPage(t, { selection: 'ref: bundle\nref: empty', done: [] });
  await ready(page);
  await isCurrent(page);
  await button(page, 'show container groups').click();
  await button(page, 'bundle').click();
  const expected = ['a_1', 'a_2', 'b_1', 'b_2', 'c_1', 's_1'];
  await assertFlatItems(page, expected);
  await button(page, 'Back to list').click();
  await button(page, 'singletons: inlined').click();
  await button(page, 'bundle').click();
  await assertFlatItems(page, expected);

  await page.setViewportSize({ width: 390, height: 844 });
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
  if (artifacts) await page.screenshot({ path: join(artifacts, 'group-items-mobile.png'), fullPage: true });

  await button(page, 'Back to list').click();
  await button(page, 'empty').click();
  await groupItems(page).getByText('This group contains no resolved items.', { exact: true }).waitFor();
  await assertFlatItems(page, []);
});

test('flat lists share packing progress and completion filters with the grouped list', async t => {
  const page = await openPage(t, { selection: 'ref: group_a', done: ['b_1'], doneDisplay: 'hidden' });
  await ready(page);
  await isCurrent(page);
  await button(page, 'group_a').click();
  await assertFlatItems(page, ['a_1', 'a_2', 'b_2']);
  await groupItems(page).getByText('b_2', { exact: true }).click();
  await assertFlatItems(page, ['a_1', 'a_2']);
  await button(page, 'view done').click();
  await assertFlatItems(page, ['b_1', 'b_2']);
  await groupItems(page).getByText('b_1', { exact: true }).click();
  await assertFlatItems(page, ['b_2']);
  await button(page, 'Back to list').click();
  await button(page, 'group_b').click();
  await assertFlatItems(page, ['b_2']);
  await groupItems(page).getByText('b_2', { exact: true }).click();
  await groupItems(page).getByText('No items done in this group.', { exact: true }).waitFor();
  await button(page, 'view todo').click();
  await assertFlatItems(page, ['b_1', 'b_2']);
  await groupItems(page).getByText('b_1', { exact: true }).click();
  await button(page, 'done: hidden').click();
  await assertFlatItems(page, ['b_2', 'b_1']);
  assert.equal(await groupItems(page).getByText('b_1', { exact: true }).evaluate(el => getComputedStyle(el).opacity), '0.45');
  assert.deepEqual((await savedState(page)).done, ['b_1']);

  await page.reload();
  await ready(page);
  await isCurrent(page);
  assert.equal(await button(page, 'Back to list').count(), 0);
  await button(page, 'group_a').click();
  await assertFlatItems(page, ['a_1', 'a_2', 'b_2', 'b_1']);
});

test('group lists respect Current and Proposed snapshots and reset navigation on Apply and Discard', async t => {
  const committed = {
    selection: 'ref: group_a',
    groups: [{ name: 'group_a', contents: { refs: [], items: ['Old item'] } }],
  };
  const page = await openPage(t, { version: 2, committed, draftText: committed.selection, done: ['b_1'], doneDisplay: 'hidden' });
  await ready(page);
  await button(page, 'Current list').click();
  await button(page, 'group_a').click();
  await assertFlatItems(page, ['Old item']);
  await button(page, 'Proposed list').click();
  await button(page, 'Back to list').waitFor({ state: 'hidden' });
  assert.equal(await button(page, 'Back to list').count(), 0);
  await button(page, 'group_a').click();
  await assertFlatItems(page, ['a_1', 'a_2', 'b_2', 'b_1']);
  await groupItems(page).getByText('b_2', { exact: true }).click();
  assert.deepEqual((await savedState(page)).done, ['b_1']);
  assert.equal(await button(page, 'done: hidden').count(), 0);
  await button(page, 'Discard changes').click();
  await isCurrent(page);
  assert.equal(await button(page, 'Back to list').count(), 0);
  await button(page, 'group_a').click();
  await assertFlatItems(page, ['Old item']);

  await button(page, 'Refresh preview').click();
  await ready(page);
  await button(page, 'Proposed list').click();
  await button(page, 'group_a').click();
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  assert.equal(await button(page, 'Back to list').count(), 0);
  await button(page, 'group_a').click();
  await assertFlatItems(page, ['a_1', 'a_2', 'b_2']);
});

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

test('a missing server selection starts empty and Apply saves exact text for a fresh browser', async t => {
  const page = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(page);
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), '', 'Unrelated browser state must not seed a new trip');
  assert.equal(await readFile(selectionFile, 'utf8'), '');
  const text = '# trip notes\nref: sun\n\n  Passport\n';
  await edit(page, text);
  assert.equal(await readFile(selectionFile, 'utf8'), '', 'Previewing must not write the selection');
  let resolutions = 0;
  page.on('request', request => { if (request.url().endsWith('/selection/')) resolutions++; });
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  assert.equal(await readFile(selectionFile, 'utf8'), text);
  assert.equal((await savedState(page)).committed.selection, text);
  assert.equal(resolutions, 0, 'Saving does not resolve the accepted preview again');

  const fresh = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(fresh);
  await isCurrent(fresh);
  assert.equal(await editor(fresh).inputValue(), text);
  await fresh.getByText('Passport', { exact: true }).waitFor();
  await edit(fresh, '');
  await button(fresh, 'Apply changes').click();
  await isCurrent(fresh);
  assert.equal(await readFile(selectionFile, 'utf8'), '');
});

test('server selections retain local drafts and packing progress without writing them to the server', async t => {
  await writeFile(selectionFile, original);
  const page = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(page);
  await isCurrent(page);
  await page.getByText('Sunscreen', { exact: true }).first().click();
  await editor(page).fill('ref:');
  await page.getByText(/Preview failed:/).waitFor();
  assert.equal(await readFile(selectionFile, 'utf8'), original);
  await page.reload();
  await page.getByText(/Preview failed:/).waitFor();
  await page.getByText('Unapplied draft restored.', { exact: true }).waitFor();
  assert.equal(await editor(page).inputValue(), 'ref:');
  assert.deepEqual((await savedState(page)).done, ['Sunscreen']);
  await button(page, 'Discard changes').click();
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), original);
  assert.equal(await readFile(selectionFile, 'utf8'), original);
});

test('Apply waits for the server and keeps its draft and baseline after a failed save', async t => {
  await writeFile(selectionFile, original);
  const page = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(page);
  await edit(page, proposal);
  const baseline = (await savedState(page)).committed;
  let release;
  let started;
  const gate = new Promise(resolve => { release = resolve; });
  const saving = new Promise(resolve => { started = resolve; });
  await page.route('**/saved-selection/', async route => {
    if (route.request().method() === 'PUT') {
      started();
      await gate;
      await route.fulfill({ status: 500, body: 'Test disk write failure' });
    } else {
      await route.continue();
    }
  });
  await button(page, 'Apply changes').click();
  await saving;
  await page.getByText('Saving selection…', { exact: true }).waitFor();
  assert.equal(await editor(page).isDisabled(), true);
  assert.equal(await button(page, 'Apply changes').isDisabled(), true);
  assert.equal(await button(page, 'Discard changes').isDisabled(), true);
  assert.deepEqual((await savedState(page)).committed, baseline);
  assert.equal(await readFile(selectionFile, 'utf8'), original);
  release();
  await page.getByText(/Could not save selection:.*Test disk write failure/).waitFor();
  assert.deepEqual((await savedState(page)).committed, baseline);
  assert.equal(await editor(page).inputValue(), proposal);
  assert.equal(await editor(page).isEnabled(), true);
  await page.unroute('**/saved-selection/');
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  assert.equal(await readFile(selectionFile, 'utf8'), proposal);
  assert.equal((await savedState(page)).committed.selection, proposal);
});

test('a failed initial load blocks edits and can be retried without overwriting the file', async t => {
  await writeFile(selectionFile, original);
  const page = await openPage(t, legacy, async page => {
    await page.route('**/saved-selection/', route => route.fulfill({ status: 500, body: 'Test read failure' }));
  }, savedOrigin);
  await page.getByText(/Could not load the saved selection:.*Test read failure/).waitFor();
  assert.equal(await editor(page).isDisabled(), true);
  assert.equal(await button(page, 'Apply initial selection').isDisabled(), true);
  assert.equal(await readFile(selectionFile, 'utf8'), original);
  await page.unroute('**/saved-selection/');
  await button(page, 'Retry loading selection').click();
  await ready(page);
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), original);
});

test('reload uses an externally changed server selection instead of a stale browser snapshot', async t => {
  await writeFile(selectionFile, original);
  const page = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(page);
  await writeFile(selectionFile, 'Passport\n# changed outside the UI');
  await page.reload();
  await ready(page);
  await isCurrent(page);
  assert.equal(await editor(page).inputValue(), 'Passport\n# changed outside the UI');
  assert.equal((await savedState(page)).committed.selection, 'Passport\n# changed outside the UI');
});

const conflictNotice = page => page.getByText('The saved selection changed since you loaded it. Your draft has been kept. Review the differences before applying again.', { exact: true });

async function waitForCommitted(page, selection) {
  await page.waitForFunction(selection => JSON.parse(localStorage.getItem(selectionStorageKey)).committed?.selection === selection, selection);
}

test('a stale Apply keeps the draft, reviews the newer saved selection, and can discard without writing', async t => {
  await writeFile(selectionFile, original);
  const first = await openPage(t, legacy, async () => {}, savedOrigin);
  const second = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(first);
  await ready(second);
  await isCurrent(second);
  await second.getByText('Sunscreen', { exact: true }).first().click();
  const latest = `${original}\nPassport`;
  const draft = `${original}\nMap`;
  await edit(second, draft);
  await edit(first, latest);
  await button(first, 'Apply changes').click();
  await isCurrent(first);
  let writes = 0;
  second.on('request', request => {
    if (request.url().endsWith('/saved-selection/') && request.method() === 'PUT') writes++;
  });
  const rejected = second.waitForResponse(response => response.url().endsWith('/saved-selection/') && response.status() === 412);
  await button(second, 'Apply changes').click();
  await rejected;
  await conflictNotice(second).waitFor();
  await waitForCommitted(second, latest);
  assert.equal(await editor(second).inputValue(), draft);
  assert.equal(await readFile(selectionFile, 'utf8'), latest);
  await second.getByText('1 item added · 1 removed · 4 retained', { exact: true }).waitFor();
  assert.deepEqual((await savedState(second)).done, ['Sunscreen']);

  await button(second, 'Current list').click();
  await second.getByText('Passport', { exact: true }).waitFor();
  assert.equal(await second.getByText('Map', { exact: true }).count(), 0);
  await button(second, 'Proposed list').click();
  await second.getByText('Map', { exact: true }).waitFor();
  assert.equal(await second.getByText('Passport', { exact: true }).count(), 0);
  await button(second, 'Discard changes').click();
  await isCurrent(second);
  assert.equal(await editor(second).inputValue(), latest);
  assert.equal(await conflictNotice(second).count(), 0);
  assert.equal(await readFile(selectionFile, 'utf8'), latest);
  assert.equal(writes, 1, 'Conflict review and Discard must not write the server file');
});

test('reviewed drafts send the latest ETag and another concurrent save triggers another conflict', async t => {
  await writeFile(selectionFile, original);
  const first = await openPage(t, legacy, async () => {}, savedOrigin);
  const second = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(first);
  await ready(second);
  const draft = `${original}\nMap`;
  await edit(second, draft);
  const etags = [];
  const matches = [];
  second.on('response', response => {
    if (response.url().endsWith('/saved-selection/') && response.status() === 200) etags.push(response.headers().etag);
  });
  second.on('request', request => {
    if (request.url().endsWith('/saved-selection/') && request.method() === 'PUT') matches.push(request.headers()['if-match']);
  });
  for (const latest of [`${original}\nPassport`, `${original}\nPassport\n# updated again`]) {
    await edit(first, latest);
    await button(first, 'Apply changes').click();
    await isCurrent(first);
    const rejected = second.waitForResponse(response => response.url().endsWith('/saved-selection/') && response.status() === 412);
    await button(second, 'Apply changes').click();
    await rejected;
    await waitForCommitted(second, latest);
    assert.equal(await editor(second).inputValue(), draft);
    assert.equal(await readFile(selectionFile, 'utf8'), latest);
  }
  // Incorporate the new item manually, then explicitly Apply the reviewed draft.
  const merged = `${draft}\nPassport`;
  await edit(second, merged);
  const accepted = second.waitForResponse(response => response.url().endsWith('/saved-selection/') && response.status() === 204);
  await button(second, 'Apply changes').click();
  await accepted;
  await isCurrent(second);
  assert.equal(await readFile(selectionFile, 'utf8'), merged);
  assert.equal(await conflictNotice(second).count(), 0);
  assert.equal(matches.length, 3);
  assert.equal(etags.length, 2);
  assert.equal(matches[1], etags[0]);
  assert.equal(matches[2], etags[1]);
  assert.notEqual(matches[0], matches[1]);
});

test('failed conflict recovery blocks Apply and preserves the draft across reload and retry', async t => {
  await writeFile(selectionFile, original);
  const page = await openPage(t, legacy, async () => {}, savedOrigin);
  await ready(page);
  const draft = `${original}\nMap`;
  const latest = `${original}\nPassport`;
  await edit(page, draft);
  await writeFile(selectionFile, latest);
  await page.route('**/saved-selection/', route => {
    if (route.request().method() === 'GET') return route.fulfill({ status: 500, body: 'Test conflict load failure' });
    return route.continue();
  });
  await button(page, 'Apply changes').click();
  await page.getByText(/Could not load the latest selection for review:.*Test conflict load failure/).waitFor();
  assert.equal(await button(page, 'Apply changes').isDisabled(), true);
  assert.equal(await button(page, 'Discard changes').isDisabled(), true);
  assert.equal(await editor(page).inputValue(), draft);
  assert.equal(await readFile(selectionFile, 'utf8'), latest);
  await page.unroute('**/saved-selection/');

  // A failed resolution must not install a new hash alongside the old baseline.
  await page.route('**/selection/', route => {
    if (route.request().postData() === latest) return route.fulfill({ status: 500, body: 'Test latest resolution failure' });
    return route.continue();
  });
  await button(page, 'Retry loading latest selection').click();
  await page.getByText(/Could not load the latest selection for review:.*Test latest resolution failure/).waitFor();
  assert.equal(await button(page, 'Apply changes').isDisabled(), true);
  assert.equal((await savedState(page)).committed.selection, original);
  await page.unroute('**/selection/');

  // Reloading while recovery is unfinished must still retain the local draft.
  await page.reload();
  await waitForCommitted(page, latest);
  await ready(page);
  await conflictNotice(page).waitFor();
  assert.equal(await editor(page).inputValue(), draft);
  assert.equal(await readFile(selectionFile, 'utf8'), latest);
  await button(page, 'Apply changes').click();
  await isCurrent(page);
  assert.equal(await readFile(selectionFile, 'utf8'), draft);
});
