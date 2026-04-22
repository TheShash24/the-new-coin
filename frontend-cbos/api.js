// ── Hawal Digital — CBOS API Client (Org3MSP) ───────────────
const API_BASE = 'http://localhost:8080'
const ORG = 'Org3'

// ── Core Fetch ────────────────────────────────────────────

async function apiFetch(method, path, body = null) {
  const opts = {
    method,
    headers: { 'X-Org-ID': ORG }
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

function calcFee(amount) {
  return Math.max(1, Math.ceil(Number(amount) * 10 / 10000))
}

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

function fmtHMZ(n) {
  return Number(n).toLocaleString() + ' HMZ'
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

// ── Transaction History Table ─────────────────────────────

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
    const feeCell  = showType ? `<td>${tx.fee > 0 ? fmtAmount(tx.fee) + ' HMZ' : '—'}</td>` : ''
    const memoLine = tx.memo ? `<div style="font-size:.75rem;color:var(--neutral-500);font-style:italic;margin-top:2px">${escHtml(tx.memo)}</div>` : ''
    return `<tr>
      <td>${fmtDate(tx.timestamp)}</td>
      ${typeCell}
      <td class="tx-id" title="${escHtml(counterpart ?? '')}">${escHtml(truncId(counterpart, 18))}</td>
      <td class="activity-amount ${dir}"><strong>${sign}${fmtAmount(tx.amount)}</strong>${memoLine}</td>
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
