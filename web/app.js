'use strict';

const $ = (id) => document.getElementById(id);
const card = $('card');
const imgFront = $('img-front');
const imgBack = $('img-back');
const fmt = (n) => n.toLocaleString();

let state = null;
let busy = false;
let celebrated = false;
const preloaded = new Map();

async function api(path, body) {
  const res = await fetch(path, body === undefined ? {} : {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({ error: 'Lost connection to the computer' }));
  return { ok: res.ok, status: res.status, data };
}

function preload(url) {
  if (!url || preloaded.has(url)) return;
  const im = new Image();
  im.src = url;
  preloaded.set(url, im);
  if (preloaded.size > 12) preloaded.delete(preloaded.keys().next().value);
}

function setImg(el, url) {
  if (!url) { el.removeAttribute('src'); return; }
  if (el.getAttribute('src') !== url) el.src = url;
}

function showMessage(html) {
  const m = $('message');
  m.innerHTML = html || '';
  m.hidden = !html;
  if (html) $('info').innerHTML = '';
  card.hidden = !!html;
  document.querySelector('.card.back').hidden = !!html;
}

function render(s) {
  state = s;
  $('setup-link').hidden = !s.isLocal;
  $('n-sel').textContent = fmt(s.selected);
  $('n-target').textContent = `/ ${fmt(s.target || 0)} selected`;
  $('n-rej').textContent = fmt(s.rejected);
  $('n-may').textContent = fmt(s.maybe);
  $('n-left').textContent = fmt(s.undecided);
  const seen = s.total ? (s.selected + s.rejected) / s.total : 0;
  $('bar-seen').style.width = (seen * 100) + '%';
  $('bar-sel').style.width = Math.min(100, s.target ? s.selected / s.target * 100 : 0) + '%';
  $('btn-undo').disabled = !s.canUndo;
  document.body.classList.toggle('over-target', s.target > 0 && s.selected >= s.target);

  if (!s.configured) {
    return showMessage(s.isLocal
      ? 'Let’s get started.<br><a href="/setup">Choose your photo folder →</a>'
      : 'Almost there!<br>Finish setup on the computer that has the photos.');
  }
  if (s.error) {
    return showMessage(`⚠️ ${esc(s.error)}` + (s.isLocal ? '<br><a href="/setup">Open settings →</a>' : ''));
  }
  if (s.scanning) {
    return showMessage(`${esc(s.phase || 'Finding photos')}…<br><b>${fmt(s.scanned)}</b> so far` +
      '<br><small>First time on a big drive takes a few minutes.</small>');
  }
  if (s.total === 0) {
    return showMessage('No photos found in that folder.' + (s.isLocal ? '<br><a href="/setup">Choose another folder →</a>' : ''));
  }
  if (s.done) {
    return showMessage(`🎉 All done!<br><b>${fmt(s.selected)}</b> photos selected out of ${fmt(s.total)}.` +
      `<br><small>They’re in the “Selected” folder.</small>` +
      (s.canUndo ? '<br><small>Changed your mind? Tap ↶ to undo.</small>' : ''));
  }
  showMessage('');
  const cur = s.current;
  if (imgFront.getAttribute('src') !== cur.url) {
    $('no-preview').hidden = true;
    setImg(imgFront, cur.url);
  }
  const next = s.upcoming && s.upcoming[0];
  setImg(imgBack, next && next.url);
  (s.upcoming || []).forEach((p) => preload(p.url));
  $('info').innerHTML =
    `${cur.maybe ? '<span class="tag">2nd look</span> ' : ''}<b>${esc(cur.name)}</b>` +
    `${cur.folder ? ' · ' + esc(cur.folder) : ''}${cur.date ? ' · ' + cur.date : ''}` +
    ` <span class="pos">#${fmt(cur.position)} of ${fmt(s.total)}</span>`;

  if (!celebrated && s.target && s.selected === s.target && s.last && s.last.op === 'select') {
    celebrated = true;
    toast(`🎉 ${fmt(s.target)} selected — goal reached!`, 'yes', 2500);
  }
}

function esc(t) {
  return String(t).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

let toastTimer;
function toast(text, kind, ms = 900) {
  const t = $('toast');
  t.textContent = text;
  t.className = 'toast show ' + (kind || '');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { t.className = 'toast'; }, ms);
}

async function refresh() {
  const r = await api('/api/state');
  if (r.ok) render(r.data);
}

const OUT = { select: [1, 0], reject: [-1, 0], maybe: [0, -1] };

async function decide(op) {
  if (busy || !state || !state.current) return;
  busy = true;
  const cur = state.current;
  const next = state.upcoming && state.upcoming[0];
  const [x, y] = OUT[op];
  card.style.transition = 'transform .28s ease-in, opacity .28s ease-in';
  card.style.transform = `translate(${x * 130}vw, ${y * 120}vh) rotate(${x * 25}deg)`;
  card.style.opacity = '0';
  const label = { select: '♥ Selected', reject: '✕ Rejected', maybe: '? Maybe later' }[op];
  toast(op === 'select' ? `${label} · ${fmt(state.selected + 1)}/${fmt(state.target)}` : label,
    { select: 'yes', reject: 'no', maybe: 'maybe' }[op]);

  const req = api('/api/decide', { path: cur.path, op });
  await new Promise((r) => setTimeout(r, 280));
  // Show the next photo (already loaded behind) while the server catches up.
  resetCard(false);
  if (next) { $('no-preview').hidden = true; setImg(imgFront, next.url); }
  const r = await req;
  busy = false;
  if (r.data && r.data.total !== undefined) render(r.data);
  if (!r.ok && r.status !== 409) toast('⚠️ ' + (r.data.error || 'Something went wrong'), 'no', 4000);
}

async function undo() {
  if (busy || !state || !state.canUndo) return;
  busy = true;
  const r = await api('/api/undo', {});
  busy = false;
  if (r.ok) {
    render(r.data);
    toast('↶ Undone');
    card.style.transition = 'none';
    card.style.transform = 'scale(.92)';
    requestAnimationFrame(() => { card.style.transition = 'transform .2s'; card.style.transform = ''; });
  } else {
    toast('⚠️ ' + (r.data.error || 'Could not undo'), 'no', 4000);
  }
}

function resetCard(animate) {
  card.style.transition = animate ? 'transform .2s ease-out' : 'none';
  card.style.transform = '';
  card.style.opacity = '';
  setStamps(0, 0);
}

function setStamps(dx, dy) {
  const w = card.offsetWidth || 300;
  const up = dy < 0 && Math.abs(dy) > Math.abs(dx);
  $('stamp-yes').style.opacity = !up && dx > 0 ? Math.min(1, dx / (w * 0.25)) : 0;
  $('stamp-no').style.opacity = !up && dx < 0 ? Math.min(1, -dx / (w * 0.25)) : 0;
  $('stamp-maybe').style.opacity = up ? Math.min(1, -dy / 150) : 0;
}

// Drag / swipe handling
let drag = null;
card.addEventListener('pointerdown', (e) => {
  if (busy || !state || !state.current) return;
  drag = { x: e.clientX, y: e.clientY, t: performance.now(), dx: 0, dy: 0 };
  card.setPointerCapture(e.pointerId);
  card.style.transition = 'none';
});
card.addEventListener('pointermove', (e) => {
  if (!drag) return;
  drag.dx = e.clientX - drag.x;
  drag.dy = e.clientY - drag.y;
  card.style.transform = `translate(${drag.dx}px, ${Math.min(drag.dy, 40)}px) rotate(${drag.dx / 18}deg)`;
  setStamps(drag.dx, drag.dy);
});
function endDrag() {
  if (!drag) return;
  const { dx, dy, t } = drag;
  drag = null;
  const w = card.offsetWidth || 300;
  const fast = Math.abs(dx) / Math.max(1, performance.now() - t) > 0.6;
  const up = dy < 0 && Math.abs(dy) > Math.abs(dx);
  if (up && -dy > 120) return decide('maybe');
  if (!up && (Math.abs(dx) > w * 0.28 || (fast && Math.abs(dx) > 40))) return decide(dx > 0 ? 'select' : 'reject');
  resetCard(true);
}
card.addEventListener('pointerup', endDrag);
card.addEventListener('pointercancel', endDrag);

imgFront.addEventListener('error', () => {
  if (imgFront.getAttribute('src')) $('no-preview').hidden = false;
});
imgFront.addEventListener('load', () => { $('no-preview').hidden = true; });

$('btn-yes').onclick = () => decide('select');
$('btn-no').onclick = () => decide('reject');
$('btn-maybe').onclick = () => decide('maybe');
$('btn-undo').onclick = undo;

document.addEventListener('keydown', (e) => {
  if (e.metaKey || e.ctrlKey || e.altKey) return;
  const k = e.key;
  if (k === 'ArrowRight') decide('select');
  else if (k === 'ArrowLeft') decide('reject');
  else if (k === 'ArrowUp') decide('maybe');
  else if (k === 'z' || k === 'Z' || k === 'Backspace') undo();
  else return;
  e.preventDefault();
});

// Keep the phone screen on while casting.
let wakeLock = null;
async function keepAwake() {
  try {
    if ('wakeLock' in navigator && !wakeLock) {
      wakeLock = await navigator.wakeLock.request('screen');
      wakeLock.addEventListener('release', () => { wakeLock = null; });
    }
  } catch (_) { /* not supported or not allowed yet */ }
}
document.addEventListener('pointerdown', keepAwake, { passive: true });
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') { keepAwake(); refresh(); }
});

// Stay in sync if someone swipes on another device (or setup changes).
setInterval(() => {
  if (!busy && !drag && document.visibilityState === 'visible') {
    api('/api/state').then((r) => {
      if (!r.ok || busy || drag) return;
      const s = r.data;
      const changed = !state || s.scanning || state.scanning !== s.scanning || s.configured !== state.configured ||
        (s.current && s.current.path) !== (state.current && state.current.path) ||
        s.selected !== state.selected || s.rejected !== state.rejected || s.error !== state.error;
      if (changed) render(s);
    }).catch(() => {});
  }
}, 2000);

refresh().catch(() => showMessage('Can’t reach the computer.<br>Is Photo Select still running?'));
