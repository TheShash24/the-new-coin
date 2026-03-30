// ── Diaspora Platform — Shared API Client ─────────────────
// Used by all dashboard pages. No rendering logic here.

const API_BASE = 'http://localhost:8080'

// ── Core Fetch ────────────────────────────────────────────

/**
 * apiFetch — wraps fetch with X-Org-ID header and error handling.
 * @param {string} method  HTTP method
 * @param {string} path    API path (e.g. '/wallets/abc')
 * @param {string} org     'Org1' | 'Org2' | 'Org3'
 * @param {object|null} body  JSON body (numbers must already be strings)
 * @returns {{ ok: boolean, status: number, data: any }}
 */
async function apiFetch(method, path, org, body = null) {
  const opts = {
    method,
    headers: { 'X-Org-ID': org }
  }
  if (body !== null) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  try {
    const res = await fetch(API_BASE + path, opts)
    const text = await res.text()
    let data = text
    try { data = JSON.parse(text) } catch { /* keep as string */ }
    return { ok: res.ok, status: res.status, data }
  } catch (err) {
    // Network-level failure (API not running, CORS etc.)
    throw new Error('Cannot reach the API. Make sure the server is running on port 8080.')
  }
}

// ── Error Message Mapper ──────────────────────────────────

/**
 * friendlyError — translates raw API error strings to user-facing messages.
 * Keeps CBOS/admin pages somewhat technical while protecting end-users.
 */
function friendlyError(data) {
  const raw = (typeof data === 'string' ? data : (data?.error ?? '')).toLowerCase()
  if (!raw) return 'Something went wrong. Please try again.'
  if (raw.includes('not found') || raw.includes('does not exist')) return 'Wallet not found. Check the ID and try again.'
  if (raw.includes('frozen'))         return 'This wallet is frozen and cannot send or receive.'
  if (raw.includes('insufficient') || raw.includes('balance'))  return 'Insufficient balance for this transaction.'
  if (raw.includes('kyc') || raw.includes('tier'))  return 'This amount exceeds your KYC tier limit.'
  if (raw.includes('role'))           return 'This operation is not allowed for this wallet type.'
  if (raw.includes('already exists') || raw.includes('duplicate')) return 'This wallet ID is already in use. Choose a different ID.'
  if (raw.includes('deposit') || raw.includes('ref')) return 'This deposit reference has already been used.'
  if (raw.includes('self'))           return 'You cannot send to your own wallet.'
  if (raw.includes('pending') || raw.includes('status')) return 'This burn request has already been processed.'
  if (raw.includes('x-org-id'))       return 'Authentication error. Please reload the page.'
  return 'Something went wrong. Please try again.'
}

// ── Toast Notifications ───────────────────────────────────

let _toastContainer = null

function _getToastContainer() {
  if (!_toastContainer) {
    _toastContainer = document.createElement('div')
    _toastContainer.className = 'toast-container'
    document.body.appendChild(_toastContainer)
  }
  return _toastContainer
}

/**
 * showToast — renders a notification toast that auto-dismisses after 4 s.
 * @param {string} msg    Message text
 * @param {'success'|'error'|'info'} type
 */
function showToast(msg, type = 'info') {
  const icons = { success: '✓', error: '✕', info: 'ℹ' }
  const container = _getToastContainer()

  const toast = document.createElement('div')
  toast.className = `toast ${type}`
  toast.innerHTML = `
    <span class="toast-icon">${icons[type] ?? 'ℹ'}</span>
    <span class="toast-body">${escHtml(msg)}</span>
    <button class="toast-close" aria-label="Dismiss">✕</button>`

  const close = () => {
    toast.style.opacity = '0'
    toast.style.transition = 'opacity .3s'
    setTimeout(() => toast.remove(), 300)
  }
  toast.querySelector('.toast-close').addEventListener('click', close)
  setTimeout(close, 4000)

  container.appendChild(toast)
}

// ── Modal Helpers ─────────────────────────────────────────

function openModal(id) {
  const el = document.getElementById(id)
  if (el) el.classList.add('open')
}

function closeModal(id) {
  const el = document.getElementById(id)
  if (el) el.classList.remove('open')
}

// Close modal on overlay click (not on modal content click)
document.addEventListener('click', e => {
  if (e.target.classList.contains('modal-overlay')) {
    e.target.classList.remove('open')
  }
})

// ── LocalStorage Helpers ──────────────────────────────────

function saveItem(key, value) {
  try { localStorage.setItem(key, JSON.stringify(value)) } catch { /* storage full */ }
}

function loadItem(key, fallback = null) {
  try {
    const v = localStorage.getItem(key)
    return v !== null ? JSON.parse(v) : fallback
  } catch { return fallback }
}

// ── Utility ───────────────────────────────────────────────

function escHtml(s) {
  return String(s)
    .replace(/&/g,'&amp;')
    .replace(/</g,'&lt;')
    .replace(/>/g,'&gt;')
    .replace(/"/g,'&quot;')
}

function fmtDate(isoStr) {
  if (!isoStr) return '—'
  try {
    return new Date(isoStr).toLocaleString(undefined, {
      year:'numeric', month:'short', day:'numeric',
      hour:'2-digit', minute:'2-digit'
    })
  } catch { return isoStr }
}

function fmtAmount(n) {
  return Number(n).toLocaleString()
}

function truncId(id, len = 16) {
  if (!id) return '—'
  return id.length > len ? id.slice(0, len) + '…' : id
}

function txTypePill(type) {
  const t = (type ?? '').toLowerCase().replace('_','')
  return `<span class="pill pill-${t}">${escHtml(type ?? '—')}</span>`
}

function statusPill(status) {
  const s = (status ?? '').toLowerCase()
  return `<span class="pill pill-${s}">${escHtml(status ?? '—')}</span>`
}

// ── Tab Switcher (used on sender/recipient/vendor pages) ──

function initTabs() {
  document.querySelectorAll('.nav-tab').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.nav-tab').forEach(b => b.classList.remove('active'))
      document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'))
      btn.classList.add('active')
      const panel = document.getElementById('panel-' + btn.dataset.tab)
      if (panel) panel.classList.add('active')
    })
  })
}

// ── Sidebar Switcher (used on CBOS page) ─────────────────

function initSidebar() {
  document.querySelectorAll('.sidebar-link').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.sidebar-link').forEach(b => b.classList.remove('active'))
      document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'))
      btn.classList.add('active')
      const panel = document.getElementById('panel-' + btn.dataset.panel)
      if (panel) panel.classList.add('active')
    })
  })
}

// ── Build Transaction History Table ──────────────────────

/**
 * renderTxTable — renders a transaction history table into container.
 * @param {HTMLElement} container
 * @param {Array} records
 * @param {string} myWalletId  — used to determine direction (in/out)
 */
function renderTxTable(container, records, myWalletId) {
  if (!records || !records.length) {
    container.innerHTML = `
      <div class="empty-state">
        <div class="empty-icon">📋</div>
        <p>No transactions yet.</p>
      </div>`
    return
  }

  const rows = records.map(tx => {
    const isOut = tx.from === myWalletId
    const counterpart = isOut ? tx.to : tx.from
    const sign = isOut ? '−' : '+'
    const signClass = isOut ? 'style="color:var(--error-text)"' : 'style="color:var(--success-text)"'
    return `<tr>
      <td>${fmtDate(tx.timestamp)}</td>
      <td>${txTypePill(tx.txType)}</td>
      <td class="tx-id" title="${escHtml(counterpart ?? '')}">${escHtml(truncId(counterpart, 18))}</td>
      <td ${signClass}><strong>${sign}${fmtAmount(tx.amount)}</strong> DSP</td>
      <td>${tx.fee > 0 ? fmtAmount(tx.fee) + ' DSP' : '—'}</td>
    </tr>`
  }).join('')

  container.innerHTML = `
    <div class="table-wrap">
      <table class="data-table">
        <thead><tr>
          <th>Date</th><th>Type</th><th>Counterpart Wallet</th>
          <th>Amount</th><th>Fee</th>
        </tr></thead>
        <tbody>${rows}</tbody>
      </table>
    </div>`
}
