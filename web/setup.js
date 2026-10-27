'use strict';

const $ = (id) => document.getElementById(id);
const form = $('flow');
let outputTouched = false;
let step = 1;
let latest = null;

function show(id, on) { $(id).hidden = !on; }
function n(x) { return (x || 0).toLocaleString(); }

function showError(msg) {
  $('error').textContent = msg || '';
  show('error', !!msg);
}

function goto(s) {
  step = s;
  document.querySelectorAll('.step').forEach((el) => { el.hidden = +el.dataset.step !== s; });
  document.querySelectorAll('#stepper li').forEach((li) => {
    const k = +li.dataset.s;
    li.classList.toggle('current', k === s);
    li.classList.toggle('done', k < s);
  });
  showError('');
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function enterFlow(s) {
  show('home', false);
  show('flow', true);
  goto(s);
}

function suggestOutput() {
  const src = $('source').value.trim().replace(/\/+$/, '');
  if (!src || outputTouched) return;
  $('output').value = src + ' - Selected';
  syncPaths();
}

function syncPaths() {
  const s = $('source').value.trim();
  const o = $('output').value.trim();
  $('source-view').textContent = s || 'No folder chosen yet';
  $('source-view').classList.toggle('set', !!s);
  $('output-view').textContent = o || '—';
  $('output-view').classList.toggle('set', !!o);
}

function setTarget(v) {
  $('target').value = v;
  document.querySelectorAll('.chip').forEach((c) => c.classList.toggle('on', +c.dataset.n === +v));
}

function fill(cfg) {
  if (!cfg) return;
  $('source').value = cfg.source || '';
  $('output').value = cfg.output || '';
  outputTouched = !!cfg.output;
  setTarget(cfg.target || 500);
  document.querySelectorAll('input[name=selectMode]').forEach((r) => { r.checked = r.value === (cfg.selectMode || 'copy'); });
  document.querySelectorAll('input[name=rejectMode]').forEach((r) => { r.checked = r.value === (cfg.rejectMode || 'none'); });
  syncPaths();
}

function statChips(s) {
  return [
    ['sel', s.selected, 'kept'], ['rej', s.rejected, 'skipped'], ['may', s.maybe, 'maybe'], ['', s.undecided, 'to go'],
  ].map(([c, v, l]) => `<div class="stat ${c}"><b>${n(v)}</b><span>${l}</span></div>`).join('');
}

function showStatus(s) {
  latest = s;
  const ok = s.configured && !s.error;
  show('foot', true);
  $('urls').innerHTML = (s.urls || []).map((u) => `<div><span>${u}</span><button type="button" class="copy" data-copy="${u}">Copy</button></div>`).join('') ||
    '<small>Couldn’t find a Wi-Fi address. Is this computer connected to Wi-Fi?</small>';

  const scan = $('scan');
  if (s.scanning) {
    scan.classList.add('busy');
    $('scan-line').textContent = `${s.phase || 'Finding photos'}… ${n(s.scanned)} so far`;
  } else if (ok) {
    scan.classList.remove('busy');
    $('scan-bar').style.width = '100%';
    $('scan-line').textContent = `✓ Ready: ${n(s.total)} photos` + (s.hidden ? ` (${n(s.hidden)} duplicates skipped)` : '');
  }
  $('home-line').textContent = s.scanning ? 'Getting your photos ready…' : `${n(s.total)} photos in your library`;
  $('home-stats').innerHTML = statChips(s);
  if (s.error) showError(s.error);
}

async function load() {
  const s = await (await fetch('/api/state')).json();
  if (!s.isLocal) { show('not-local', true); $('port').textContent = location.port; return; }
  fill(s.config);
  showStatus(s);
  if (s.configured && !s.error) { show('home', true); if (s.scanning) setTimeout(poll, 1000); }
  else { enterFlow(1); if (s.error) showError(s.error); }
}

async function poll() {
  const s = await (await fetch('/api/state')).json();
  showStatus(s);
  if (s.scanning) setTimeout(poll, 1000);
}

function checkStep() {
  if (step === 1 && !$('source').value.trim()) { showError('Choose the folder that has your photos first.'); return false; }
  if (step === 2 && !$('output').value.trim()) { showError('Choose where your picks should go.'); return false; }
  return true;
}

document.querySelectorAll('[data-next]').forEach((b) => b.addEventListener('click', () => { if (checkStep()) goto(step + 1); }));
document.querySelectorAll('[data-back]').forEach((b) => b.addEventListener('click', () => goto(step - 1)));
document.querySelectorAll('[data-goto]').forEach((b) => b.addEventListener('click', () => goto(+b.dataset.goto)));
$('home-change').addEventListener('click', () => enterFlow(1));
$('home-phone').addEventListener('click', () => enterFlow(4));

document.querySelectorAll('.chip').forEach((c) => c.addEventListener('click', () => setTarget(c.dataset.n)));
$('target').addEventListener('input', () => setTarget($('target').value));

$('source').addEventListener('input', () => { suggestOutput(); syncPaths(); });
$('output').addEventListener('input', () => { outputTouched = true; syncPaths(); });

document.addEventListener('click', (e) => {
  const b = e.target.closest('.copy');
  if (!b) return;
  navigator.clipboard?.writeText(b.dataset.copy).then(() => { b.textContent = 'Copied ✓'; setTimeout(() => { b.textContent = 'Copy'; }, 1500); });
});

document.querySelectorAll('.pick').forEach((btn) => {
  btn.addEventListener('click', async () => {
    btn.disabled = true;
    try {
      const r = await fetch('/api/pick-folder', { method: 'POST' });
      const d = await r.json();
      if (!r.ok) { showError(d.error + ' — open “Or type the folder path” below and paste it instead.'); return; }
      if (d.path) {
        $(btn.dataset.target).value = d.path;
        if (btn.dataset.target === 'output') outputTouched = true;
        else suggestOutput();
        syncPaths();
        showError('');
      }
    } finally {
      btn.disabled = false;
    }
  });
});

$('start').addEventListener('click', async () => {
  showError('');
  $('start').disabled = true;
  try {
    const body = {
      source: $('source').value,
      output: $('output').value,
      target: parseInt($('target').value, 10) || 500,
      selectMode: document.querySelector('input[name=selectMode]:checked').value,
      rejectMode: document.querySelector('input[name=rejectMode]:checked').value,
    };
    const r = await fetch('/api/setup', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
    });
    const d = await r.json();
    if (!r.ok) {
      showError(d.error);
      goto(/folder|photo/i.test(d.error || '') ? 1 : 2);
      showError(d.error);
      return;
    }
    fill(d.config);
    showStatus(d);
    goto(4);
    setTimeout(poll, 800);
  } catch (err) {
    showError('Couldn’t reach Photo Select. Is it still running?');
  } finally {
    $('start').disabled = false;
  }
});

$('quit').addEventListener('click', async () => {
  await fetch('/api/quit', { method: 'POST' }).catch(() => {});
  document.querySelector('.wizard').innerHTML =
    '<header class="brand"><img src="/static/icon.svg" alt="" width="52" height="52"><div><h1>Photo Select has stopped</h1><p>You can close this tab.</p></div></header>' +
    '<section class="card"><h2>All saved 👍</h2><p class="sub">Open Photo Select again whenever you want to carry on where you left off.</p></section>';
});

load();
