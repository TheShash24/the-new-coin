// ── Hawal Digital — Shared API Client ──────────────────────
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
    throw new Error('Cannot reach the API. Make sure the server is running on port 8080.')
  }
}

// ── Error Message Mapper ──────────────────────────────────

/**
 * friendlyError — translates raw API error strings to consumer-friendly messages.
 */
function friendlyError(data) {
  const raw = (typeof data === 'string' ? data : (data?.error ?? '')).toLowerCase()
  if (!raw) return 'Something went wrong. Please try again.'
  if (raw.includes('not found') || raw.includes('does not exist')) return 'Account not found. Check the number and try again.'
  if (raw.includes('frozen'))                                       return 'This account has been restricted. Please contact support.'
  if (raw.includes('insufficient') || raw.includes('balance'))      return "You don't have enough funds for this transfer."
  if (raw.includes('kyc') || raw.includes('tier'))                  return 'This amount exceeds your verification limit.'
  if (raw.includes('role'))                                         return "This action isn't available for your account type."
  if (raw.includes('already exists') || raw.includes('duplicate'))  return 'This account number is already in use.'
  if (raw.includes('deposit') || raw.includes('ref'))               return 'This deposit reference has already been used.'
  if (raw.includes('self'))                                         return "You can't send to your own account."
  if (raw.includes('pending') || raw.includes('status'))            return 'This request has already been processed.'
  if (raw.includes('x-org-id'))                                     return 'Authentication error. Please reload the page.'
  return 'Something went wrong. Please try again.'
}

// ── Fee & Currency Helpers ────────────────────────────────

/**
 * calcFee — mirrors chaincode calculateFee (10 basis points, min 1).
 * @param {number|string} amount
 * @returns {number}
 */
function calcFee(amount) {
  return Math.max(1, Math.ceil(Number(amount) * 10 / 10000))
}

/**
 * fmtCurrency — formats a number as currency with symbol prefix.
 * @param {number|string} amount
 * @param {string} symbol
 * @returns {string}  e.g. "£1,234.50"
 */
function fmtCurrency(amount, symbol = '£') {
  return symbol + Number(amount).toLocaleString('en-GB', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  })
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
    return new Date(isoStr).toLocaleString('en-GB', {
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

const STATUS_MAP = {
  pending:  { cls: 'processing', label: 'Processing' },
  approved: { cls: 'approved',   label: 'Approved'   },
  rejected: { cls: 'declined',   label: 'Declined'   },
}

function statusPill(status) {
  const entry = STATUS_MAP[(status ?? '').toLowerCase()]
  const cls   = entry ? entry.cls   : (status ?? '').toLowerCase()
  const label = entry ? entry.label : (status ?? '—')
  return `<span class="pill pill-${cls}">${escHtml(label)}</span>`
}

// ── Tab Switcher ──────────────────────────────────────────

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

// ── Sidebar Switcher (CBOS page) ──────────────────────────

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

// ── Transaction History Table (CBOS / admin use) ──────────

/**
 * renderTxTable — renders a full transaction table.
 * @param {HTMLElement} container
 * @param {Array} records
 * @param {string} myWalletId
 * @param {boolean} showType  — pass false on consumer pages
 */
function renderTxTable(container, records, myWalletId, showType = true) {
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
    const dir      = isOut ? 'out' : 'in'
    const sign     = isOut ? '−' : '+'
    const typeCell = showType ? `<td>${txTypePill(tx.txType)}</td>` : ''
    const feeCell  = showType ? `<td>${tx.fee > 0 ? fmtAmount(tx.fee) + ' DSP' : '—'}</td>` : ''
    return `<tr>
      <td>${fmtDate(tx.timestamp)}</td>
      ${typeCell}
      <td class="tx-id" title="${escHtml(counterpart ?? '')}">${escHtml(truncId(counterpart, 18))}</td>
      <td class="activity-amount ${dir}"><strong>${sign}${fmtAmount(tx.amount)}</strong></td>
      ${feeCell}
    </tr>`
  }).join('')

  const typeHead = showType ? '<th>Type</th>' : ''
  const feeHead  = showType ? '<th>Fee</th>'  : ''

  container.innerHTML = `
    <div class="table-wrap">
      <table class="data-table">
        <thead><tr>
          <th>Date</th>${typeHead}<th>Account</th>
          <th>Amount</th>${feeHead}
        </tr></thead>
        <tbody>${rows}</tbody>
      </table>
    </div>`
}

// ── Activity List (consumer pages — card style) ───────────

/**
 * renderActivityList — card-list variant for recent activity on consumer home tabs.
 * @param {HTMLElement} container
 * @param {Array} records
 * @param {string} myWalletId
 */
function renderActivityList(container, records, myWalletId) {
  if (!records || !records.length) {
    container.innerHTML = `
      <div class="empty-state">
        <div class="empty-icon">📋</div>
        <p>No recent activity.</p>
      </div>`
    return
  }

  container.innerHTML = records.slice(0, 5).map(tx => {
    const isOut       = tx.from === myWalletId
    const dir         = isOut ? 'out' : 'in'
    const counterpart = isOut ? (tx.to ?? '—') : (tx.from ?? '—')
    const sign        = isOut ? '−' : '+'
    const icon        = isOut ? '↑' : '↓'

    return `
      <div class="activity-item">
        <div class="activity-icon ${dir}">${icon}</div>
        <div class="activity-body">
          <div class="activity-title">${escHtml(truncId(counterpart, 20))}</div>
          <div class="activity-date">${fmtDate(tx.timestamp)} · <span class="pill pill-completed">Completed</span></div>
        </div>
        <div class="activity-right">
          <div class="activity-amount ${dir}">${sign}${fmtCurrency(tx.amount)}</div>
        </div>
      </div>`
  }).join('')
}
