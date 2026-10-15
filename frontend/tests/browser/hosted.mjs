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
const test = (name, run) => nodeTest(name, { timeout: 60000 }, run);
let temporary, binary, browser;
before(async () => {
  temporary = await mkdtemp(join(tmpdir(), 'packing-hosted-browser-'));
  execFileSync('make', ['frontend'], { cwd: repo, stdio: 'inherit' });
  binary = join(temporary, 'packing-server');
  execFileSync('go', ['build', '-o', binary, './cmd/packing-server'], { cwd: repo, stdio: 'inherit' });
  browser = await chromium.launch();
}, { timeout: 120000 });
after(async () => { await browser?.close(); if (temporary) await rm(temporary, { recursive: true, force: true }); });
const cli = (store, ...args) => execFileSync(binary, ['hosted', '--store', store, ...args], { encoding: 'utf8' });
async function freeOrigin() {
  const probe = createServer(); probe.listen(0, '127.0.0.1'); await once(probe, 'listening');
  const { port } = probe.address(); await new Promise(resolve => probe.close(resolve));
  return `http://127.0.0.1:${port}`;
}
async function start(store, origin) {
  const child = spawn(binary, ['hosted', '--store', store, 'serve', '--origin', origin, '--listen', new URL(origin).host, '--insecure-localhost'], { stdio: ['ignore', 'ignore', 'pipe'] });
  let diagnostic = ''; child.stderr.on('data', data => { diagnostic += data; });
  for (let i = 0; i < 100; i++) {
    if (child.exitCode !== null) throw new Error(`Server exited: ${diagnostic}`);
    try { if ((await fetch(`${origin}/health/ready`)).ok) return child; } catch {}
    await delay(25); // Only startup polling; workflow races use request gates.
  }
  throw new Error('Hosted fixture did not become ready');
}
async function stop(child) {
  if (child && child.exitCode === null && child.signalCode === null) { const exited = once(child, 'exit'); child.kill('SIGTERM'); await exited; }
}
async function fixture(t) {
  const root = await mkdtemp(join(temporary, 'case-')), store = join(root, 'store'), groups = join(root, 'groups');
  await mkdir(groups); await writeFile(join(groups, 'kit'), 'Tent\nWater\n');
  const selection = join(root, 'selection'); await writeFile(selection, 'ref: kit\n');
  cli(store, 'init', '--groups-dir', groups, '--selection-file', selection);
  const origin = await freeOrigin(), linkPath = join(root, 'access');
  cli(store, 'access', '--origin', origin, '--output', linkPath);
  const link = (await readFile(linkPath, 'utf8')).trim();
  const f = { root, store, groups, origin, link, server: await start(store, origin) };
  t.after(() => stop(f.server));
  return f;
}
const button = (page, name) => page.getByRole('button', { name, exact: true });
const editor = page => page.getByRole('textbox', { name: 'Draft selection' });
const saved = page => page.getByRole('status').filter({ hasText: /^Saved$/ }).waitFor();
const preview = page => page.getByText('Preview ready.', { exact: true }).waitFor();
const conflict = page => page.getByRole('status').filter({ hasText: /^Conflicting/ }).waitFor();
async function open(t, f, mobile = false) {
  const context = await browser.newContext({ viewport: mobile ? { width: 390, height: 844 } : { width: 1280, height: 900 }, isMobile: mobile, hasTouch: mobile });
  t.after(() => context.close());
  const page = await context.newPage(); page.setDefaultTimeout(10000);
  const errors = []; page.on('pageerror', error => errors.push(error.message));
  t.after(() => assert.deepEqual(errors, [], 'No browser runtime errors'));
  await page.goto(f.link); await saved(page); await preview(page);
  assert.equal(new URL(page.url()).hash, '', 'access secret removed from the address bar');
  return page;
}
async function state(page) {
  return page.evaluate(async () => (await fetch(`/api/v1${location.pathname}`)).json());
}
async function apply(page) {
  await preview(page);
  const response = page.waitForResponse(r => r.url().endsWith('/apply') && r.request().method() === 'POST');
  await button(page, 'Apply changes').click(); assert.equal((await response).status(), 200); await saved(page);
  await button(page, 'Current list').click();
}
async function edit(page, text) {
  const response = page.waitForResponse(r => r.url().endsWith('/draft') && r.request().method() === 'PUT');
  await editor(page).fill(text); const result = await response; return result.status();
}
async function check(page, name) {
  await button(page, 'Current list').click();
  const response = page.waitForResponse(r => r.url().includes('/items/') && r.request().method() === 'PUT');
  await page.getByText(name, { exact: true }).click(); return (await response).status();
}
async function refresh(page) { const response = page.waitForResponse(r => r.request().method() === 'GET' && /\/api\/v1\/sessions\/[^/]+$/.test(r.url())); await button(page, 'Refresh trip').click(); await response; await saved(page); }
const packed = (trip, name) => Object.values(trip.items).find(item => item.name === name)?.packed;


test('desktop/mobile round trip, pinned definitions, restart and independent restore', async t => {
  const f = await fixture(t), desktop = await open(t, f);
  assert.equal(await edit(desktop, '# Exact draft\nref: kit\nCoat\n'), 200); await saved(desktop); await apply(desktop);
  assert.equal(await check(desktop, 'Tent'), 200); await saved(desktop);
  const before = await state(desktop);
  await desktop.close(); // The server, not the original browser, owns the trip.
  const phone = await open(t, f, true);
  assert.equal(await editor(phone).inputValue(), before.draft);
  assert.deepEqual((await state(phone)).snapshot, before.snapshot);
  assert.equal(packed(await state(phone), 'Tent'), true);
  assert.equal(await check(phone, 'Tent'), 200); await saved(phone);
  assert.equal(await check(phone, 'Water'), 200); await saved(phone);
  const returned = await open(t, f);
  assert.equal(packed(await state(returned), 'Tent'), false);
  assert.equal(packed(await state(returned), 'Water'), true);
  await writeFile(join(f.groups, 'kit'), 'Changed source after provisioning');
  await button(returned, 'Refresh preview').click(); await preview(returned);
  assert.equal(await returned.getByText('Changed source after provisioning', { exact: true }).count(), 0);
  const acknowledged = await state(returned);
  await stop(f.server); f.server = await start(f.store, f.origin);
  await returned.reload(); await saved(returned); assert.deepEqual(await state(returned), acknowledged);
  await stop(f.server);
  const backup = join(f.root, 'backup.json'); cli(f.store, 'backup', '--output', backup);
  const restoredStore = join(f.root, 'restore'); cli(restoredStore, 'restore', '--input', backup);
  const restoredOrigin = await freeOrigin(), restoredLink = join(f.root, 'restore-link');
  cli(restoredStore, 'access', '--origin', restoredOrigin, '--output', restoredLink);
  const restored = { store: restoredStore, origin: restoredOrigin, link: (await readFile(restoredLink, 'utf8')).trim(), server: await start(restoredStore, restoredOrigin) };
  t.after(() => stop(restored.server));
  const recovered = await open(t, restored, true), after = await state(recovered);
  assert.notEqual(after.epoch, acknowledged.epoch); after.epoch = acknowledged.epoch;
  assert.deepEqual(after, acknowledged);
  assert.equal((await fetch(`${restoredOrigin}/api/v1${new URL(restored.link).pathname}`)).status, 401);
});

test('concurrent items survive, stale item and draft edits require explicit recovery', async t => {
  const f = await fixture(t), desktop = await open(t, f); await apply(desktop);
  const phone = await open(t, f, true);
  const codes = await Promise.all([check(desktop, 'Tent'), check(phone, 'Water')]); assert.deepEqual(codes, [200, 200]);
  await Promise.all([saved(desktop), saved(phone)]); await Promise.all([refresh(desktop), refresh(phone)]);
  assert.equal(packed(await state(desktop), 'Tent'), true); assert.equal(packed(await state(phone), 'Water'), true);
  assert.equal(await check(desktop, 'Tent'), 200); await saved(desktop);
  assert.equal(await check(phone, 'Tent'), 412); await conflict(phone);
  await button(phone, 'Discard device edits and use shared trip').click(); await saved(phone);
  assert.equal(packed(await state(phone), 'Tent'), false);
  assert.equal(await edit(desktop, 'ref: kit\nDesktop addition'), 200); await saved(desktop);
  assert.equal(await edit(phone, 'ref: kit\nPhone addition'), 412); await conflict(phone);
  assert.equal(await editor(phone).inputValue(), 'ref: kit\nPhone addition');
  await phone.getByText('ref: kit\nDesktop addition', { exact: true }).waitFor();
  await button(phone, 'Save my draft over this shared draft').click(); await saved(phone);
  await refresh(desktop); assert.equal(await editor(desktop).inputValue(), 'ref: kit\nPhone addition');
});

test('lost acknowledgement survives reload and receipt retry cannot regress a newer item edit', async t => {
  const f = await fixture(t), desktop = await open(t, f); await apply(desktop);
  const phone = await open(t, f, true);
  let original;
  await desktop.route('**/items/*', async route => { original = route.request().postDataJSON(); const response = await route.fetch(); assert.equal(response.status(), 200); await route.abort(); });
  await button(desktop, 'Current list').click(); await desktop.getByText('Tent', { exact: true }).click();
  await desktop.getByRole('status').filter({ hasText: /^Unsynced/ }).waitFor();
  await refresh(phone); assert.equal(packed(await state(phone), 'Tent'), true);
  assert.equal(await check(phone, 'Tent'), 200); await saved(phone);
  await desktop.unroute('**/items/*'); await desktop.reload();
  await desktop.getByText('Recovered unsynced edit. Retry to check its acknowledgement.', { exact: true }).waitFor();
  const retried = desktop.waitForRequest(r => r.url().includes('/items/') && r.method() === 'PUT');
  await button(desktop, 'Retry unsaved edits').click(); assert.deepEqual((await retried).postDataJSON(), original); await saved(desktop);
  assert.equal(packed(await state(desktop), 'Tent'), false);
  assert.equal(await desktop.getByText('Tent', { exact: true }).evaluate(el => el.style.opacity), '');
});

test('Apply preserves surviving progress and refuses a delayed old-generation checkbox', async t => {
  const f = await fixture(t), desktop = await open(t, f); await apply(desktop);
  assert.equal(await check(desktop, 'Tent'), 200); await saved(desktop);
  const phone = await open(t, f, true);
  let release, entered;
  const gate = new Promise(resolve => { release = resolve; }), started = new Promise(resolve => { entered = resolve; });
  await phone.route('**/items/*', async route => { entered(); await gate; await route.continue(); });
  const changing = check(phone, 'Water'); await started;
  assert.equal(await edit(desktop, 'ref: kit\nCoat'), 200); await saved(desktop); await apply(desktop);
  release(); assert.equal(await changing, 409); await conflict(phone);
  const current = await state(desktop); assert.equal(packed(current, 'Tent'), true); assert.equal(packed(current, 'Water'), false); assert.equal(packed(current, 'Coat'), false);
});

test('draft edits during a delayed save and resume refresh retain pending text', async t => {
  const f = await fixture(t), desktop = await open(t, f); await apply(desktop);
  const phone = await open(t, f, true);
  let release, entered;
  const gate = new Promise(resolve => { release = resolve; }), started = new Promise(resolve => { entered = resolve; });
  let first = true;
  await desktop.route('**/draft', async route => { if (first) { first = false; entered(); await gate; } await route.continue(); });
  await editor(desktop).fill('ref: kit\nFirst text'); await started;
  await editor(desktop).fill('ref: kit\nNewest text');
  await desktop.evaluate(() => document.dispatchEvent(new Event('visibilitychange')));
  release(); await saved(desktop);
  assert.equal(await editor(desktop).inputValue(), 'ref: kit\nNewest text');
  await phone.evaluate(() => document.dispatchEvent(new Event('visibilitychange')));
  await phone.waitForFunction(() => document.querySelector('textarea')?.value === 'ref: kit\nNewest text');
  assert.equal(await edit(desktop, 'unknown: invalid draft'), 200); await saved(desktop);
  await desktop.reload(); await saved(desktop);
  assert.equal(await editor(desktop).inputValue(), 'unknown: invalid draft');
  assert.equal((await state(desktop)).snapshot.selection, 'ref: kit\n');
});

test('two tabs retain separate pending recovery and never reuse a new mutation ID', async t => {
  const f = await fixture(t), first = await open(t, f); await apply(first);
  const second = await first.context().newPage(); second.setDefaultTimeout(10000);
  await second.goto(first.url()); await saved(second); await preview(second);
  await first.route('**/items/*', route => route.abort());
  await second.route('**/items/*', route => route.abort());
  const pending = async (page, name) => {
    await button(page, 'Current list').click(); await page.getByText(name, { exact: true }).click();
    await page.getByRole('status').filter({ hasText: /^Unsynced/ }).waitFor();
  };
  await Promise.all([pending(first, 'Tent'), pending(second, 'Water')]);
  const cached = page => page.evaluate(() => JSON.parse(sessionStorage.getItem(Object.keys(sessionStorage).find(key => key.startsWith('packing:hosted:')))));
  const a = await cached(first), b = await cached(second);
  assert.equal(a.writes.length, 1); assert.equal(b.writes.length, 1);
  assert.notEqual(a.writes[0].mutationId, b.writes[0].mutationId);
  assert.notEqual(a.writes[0].item, b.writes[0].item);
  await Promise.all([first.reload(), second.reload()]);
  await Promise.all([first.getByText('Recovered unsynced edit. Retry to check its acknowledgement.', { exact: true }).waitFor(), second.getByText('Recovered unsynced edit. Retry to check its acknowledgement.', { exact: true }).waitFor()]);
  assert.deepEqual((await cached(first)).writes, a.writes); assert.deepEqual((await cached(second)).writes, b.writes);
});

test('recovery storage failure stays visible while HTTP saves and recovery export work', async t => {
  const f = await fixture(t), context = await browser.newContext(); t.after(() => context.close());
  await context.addInitScript(() => {
    const original = Storage.prototype.setItem;
    Storage.prototype.setItem = function (key, value) { if (key.startsWith('packing:hosted:')) throw new Error('synthetic quota failure'); return original.call(this, key, value); };
  });
  const page = await context.newPage(); page.setDefaultTimeout(10000); await page.goto(f.link);
  await page.getByRole('status').filter({ hasText: /^Saved on server; device recovery unavailable$/ }).waitFor();
  assert.equal(await edit(page, 'ref: kit\nExtra'), 200);
  await page.getByRole('status').filter({ hasText: /^Saved on server; device recovery unavailable$/ }).waitFor();
  assert.equal((await state(page)).draft, 'ref: kit\nExtra');
  const downloading = page.waitForEvent('download'); await button(page, 'Export device recovery').click();
  const download = await downloading; const recovery = JSON.parse(await readFile(await download.path(), 'utf8'));
  assert.equal(recovery.draft, 'ref: kit\nExtra'); assert.deepEqual(recovery.writes, []);
});
