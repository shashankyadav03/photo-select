'use strict';

const $ = (id) => document.getElementById(id);
const form = $('form');
let outputTouched = false;

function suggestOutput() {
  const src = $('source').value.trim().replace(/\/+$/, '');
  if (!src || outputTouched) return;
  $('output').value = src + ' - Selected';
}

function showError(msg) {
  $('error').textContent = msg || '';
  $('error').hidden = !msg;
}

function fill(cfg) {
  if (!cfg) return;
  $('source').value = cfg.source || '';
  $('output').value = cfg.output || '';
  outputTouched = !!cfg.output;
  $('target').value = cfg.target || 500;
  form.querySelectorAll('input[name=selectMode]').forEach((r) => { r.checked = r.value === (cfg.selectMode || 'copy'); });
  form.querySelectorAll('input[name=rejectMode]').forEach((r) => { r.checked = r.value === (cfg.rejectMode || 'none'); });
}

function showStatus(s) {
  $('connect').hidden = !s.configured || !!s.error;
  $('urls').innerHTML = (s.urls || []).map((u) => `<div>${u}</div>`).join('') ||
    '<small>Couldn’t find a Wi-Fi address. Is this computer connected to Wi-Fi?</small>';
  let line = '';
  if (s.scanning) line = `${s.phase || 'Finding photos'}… ${s.scanned.toLocaleString()} so far`;
  else if (s.configured && !s.error) {
    line = `${s.total.toLocaleString()} photos · ${s.selected.toLocaleString()} selected · ` +
      `${s.rejected.toLocaleString()} rejected · ${s.maybe.toLocaleString()} maybe · ${s.undecided.toLocaleString()} to go` +
      (s.hidden ? ` · ${s.hidden.toLocaleString()} duplicates skipped` : '');
  }
  $('progress-line').textContent = line;
  if (s.error) showError(s.error);
}

async function load() {
  const r = await fetch('/api/state');
  const s = await r.json();
  if (!s.isLocal) {
    $('not-local').hidden = false;
    $('port').textContent = location.port;
    form.hidden = true;
    return;
  }
  $('quit-row').hidden = false;
  fill(s.config);
  showStatus(s);
  if (s.scanning) setTimeout(poll, 1000);
}

async function poll() {
  const s = await (await fetch('/api/state')).json();
  showStatus(s);
  if (s.scanning) setTimeout(poll, 1000);
}

$('quit').addEventListener('click', async () => {
  await fetch('/api/quit', { method: 'POST' }).catch(() => {});
  document.querySelector('.setup-page').innerHTML =
    '<h1>📸 Photo Select has stopped</h1><p class="lead">You can close this tab. Open Photo Select again whenever you want to carry on.</p>';
});

$('source').addEventListener('input', suggestOutput);
$('output').addEventListener('input', () => { outputTouched = true; });

document.querySelectorAll('.pick').forEach((btn) => {
  btn.addEventListener('click', async () => {
    btn.disabled = true;
    try {
      const r = await fetch('/api/pick-folder', { method: 'POST' });
      const d = await r.json();
      if (!r.ok) { showError(d.error + ' — type or paste the folder path instead.'); return; }
      if (d.path) {
        $(btn.dataset.target).value = d.path;
        if (btn.dataset.target === 'output') outputTouched = true;
        else suggestOutput();
      }
    } finally {
      btn.disabled = false;
    }
  });
});

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  showError('');
  $('start').disabled = true;
  try {
    const body = {
      source: $('source').value,
      output: $('output').value,
      target: parseInt($('target').value, 10) || 500,
      selectMode: form.querySelector('input[name=selectMode]:checked').value,
      rejectMode: form.querySelector('input[name=rejectMode]:checked').value,
    };
    const r = await fetch('/api/setup', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
    });
    const d = await r.json();
    if (!r.ok) { showError(d.error); return; }
    fill(d.config);
    showStatus(d);
    $('connect').scrollIntoView({ behavior: 'smooth' });
    setTimeout(poll, 800);
  } catch (err) {
    showError('Couldn’t reach Photo Select. Is it still running?');
  } finally {
    $('start').disabled = false;
  }
});

load();
