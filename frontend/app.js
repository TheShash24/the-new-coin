// ── Config ────────────────────────────────────────────────
const API_BASE = 'http://localhost:8080'
let currentOrg = null // 'Org1' | 'Org2' | 'Org3'

// ── API Client ────────────────────────────────────────────

async function apiFetch(method, path, body = null) {
  if (!currentOrg) {
    showResponse(0, { error: 'Select an organisation first (Bank A / Bank B / CBOS).' })
    return null
  }

  const opts = {
    method,
    headers: { 'X-Org-ID': currentOrg }
  }

  if (body !== null) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }

  let status = 0
  let data = null

  try {
    const res = await fetch(API_BASE + path, opts)
    status = res.status
    const text = await res.text()
    try { data = JSON.parse(text) } catch { data = text }
    showResponse(status, data)
    return { ok: res.ok, status, data }
  } catch (err) {
    data = { error: 'Network error — is the API running on port 8080? ' + err.message }
    showResponse(0, data)
    return null
  }
}

// ── Response Panel ────────────────────────────────────────

function showResponse(status, data) {
  const badge = document.getElementById('status-badge')
  const body  = document.getElementById('response-body')

  if (status === 0) {
    badge.textContent = 'ERR'
    badge.className = 'err'
  } else {
    badge.textContent = 'HTTP ' + status
    badge.className = status >= 200 && status < 300 ? 'ok' : 'err'
  }

  body.textContent = typeof data === 'string'
    ? data
    : JSON.stringify(data, null, 2)
}

// ── Status Messages ───────────────────────────────────────

function setStatus(id, msg, isError) {
  const el = document.getElementById(id)
  if (!el) return
  el.textContent = msg
  el.className = 'status-msg visible ' + (isError ? 'error' : 'success')
}

function clearStatus(id) {
  const el = document.getElementById(id)
  if (el) el.className = 'status-msg'
}

function val(id) {
  return (document.getElementById(id)?.value ?? '').trim()
}

// ── Tab Switching ─────────────────────────────────────────

document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'))
    document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'))
    btn.classList.add('active')
    document.getElementById('tab-' + btn.dataset.tab).classList.add('active')
  })
})

// ── Role Switcher ─────────────────────────────────────────

document.querySelectorAll('.org-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.org-btn').forEach(b => b.classList.remove('active'))
    btn.classList.add('active')
    currentOrg = btn.dataset.org
    const labels = { Org1: 'Bank A (Org1)', Org2: 'Bank B (Org2)', Org3: 'CBOS (Org3)' }
    document.getElementById('active-org-label').textContent = '— ' + labels[currentOrg]
  })
})

// ── Diaspora Flow ─────────────────────────────────────────

document.getElementById('btn-d-register').addEventListener('click', async () => {
  clearStatus('st-d-register')
  const id = val('d-reg-id'), role = val('d-reg-role'), kycTier = val('d-reg-kyc')
  if (!id) return setStatus('st-d-register', 'Wallet ID is required.', true)
  const res = await apiFetch('POST', '/wallets', { id, role, kycTier })
  if (!res) return
  res.ok
    ? setStatus('st-d-register', `Wallet "${id}" registered.`, false)
    : setStatus('st-d-register', res.data?.error ?? 'Error registering wallet.', true)
})

document.getElementById('btn-d-mint').addEventListener('click', async () => {
  clearStatus('st-d-mint')
  const walletId = val('d-mint-wallet'), amount = val('d-mint-amount'), depositRef = val('d-mint-ref')
  if (!walletId || !amount || !depositRef) return setStatus('st-d-mint', 'All fields are required.', true)
  const res = await apiFetch('POST', '/tokens/mint', { walletId, amount: String(amount), depositRef })
  if (!res) return
  res.ok
    ? setStatus('st-d-mint', `Minted ${amount} tokens to "${walletId}".`, false)
    : setStatus('st-d-mint', res.data?.error ?? 'Error minting tokens.', true)
})

document.getElementById('btn-d-transfer').addEventListener('click', async () => {
  clearStatus('st-d-transfer')
  const fromId = val('d-tx-from'), toId = val('d-tx-to'), amount = val('d-tx-amount')
  if (!fromId || !toId || !amount) return setStatus('st-d-transfer', 'All fields are required.', true)
  const res = await apiFetch('POST', '/tokens/transfer', { fromId, toId, amount: String(amount) })
  if (!res) return
  res.ok
    ? setStatus('st-d-transfer', `Transferred ${amount} tokens from "${fromId}" to "${toId}".`, false)
    : setStatus('st-d-transfer', res.data?.error ?? 'Error transferring.', true)
})

// ── Sudan-Side Flow ───────────────────────────────────────

document.getElementById('btn-s-register').addEventListener('click', async () => {
  clearStatus('st-s-register')
  const id = val('s-reg-id'), role = val('s-reg-role'), kycTier = val('s-reg-kyc')
  if (!id) return setStatus('st-s-register', 'Wallet ID is required.', true)
  const res = await apiFetch('POST', '/wallets', { id, role, kycTier })
  if (!res) return
  res.ok
    ? setStatus('st-s-register', `Wallet "${id}" registered.`, false)
    : setStatus('st-s-register', res.data?.error ?? 'Error registering wallet.', true)
})

document.getElementById('btn-s-pay').addEventListener('click', async () => {
  clearStatus('st-s-pay')
  const fromId = val('s-pay-from'), toId = val('s-pay-to'), amount = val('s-pay-amount')
  if (!fromId || !toId || !amount) return setStatus('st-s-pay', 'All fields are required.', true)
  const res = await apiFetch('POST', '/tokens/pay', { fromId, toId, amount: String(amount) })
  if (!res) return
  res.ok
    ? setStatus('st-s-pay', `Paid ${amount} tokens from "${fromId}" to vendor "${toId}".`, false)
    : setStatus('st-s-pay', res.data?.error ?? 'Error processing payment.', true)
})

document.getElementById('btn-s-burn').addEventListener('click', async () => {
  clearStatus('st-s-burn')
  const vendorId = val('s-burn-vendor'), amount = val('s-burn-amount'), burnRef = val('s-burn-ref')
  if (!vendorId || !amount || !burnRef) return setStatus('st-s-burn', 'All fields are required.', true)
  const res = await apiFetch('POST', '/burns', { vendorId, amount: String(amount), burnRef })
  if (!res) return
  res.ok
    ? setStatus('st-s-burn', `Burn request initiated. Tokens escrowed. TX ID in response panel.`, false)
    : setStatus('st-s-burn', res.data?.error ?? 'Error initiating burn.', true)
})

// ── CBOS Flow ─────────────────────────────────────────────

document.getElementById('btn-c-burns').addEventListener('click', async () => {
  clearStatus('st-c-burns')
  const pageSize = val('c-burns-pagesize') || '10'
  const bookmark = val('c-burns-bookmark')
  const params = new URLSearchParams({ pageSize, bookmark })
  const res = await apiFetch('GET', '/burns?' + params)
  if (!res) return
  if (!res.ok) return setStatus('st-c-burns', res.data?.error ?? 'Error loading burn requests.', true)

  const records = res.data?.records ?? (Array.isArray(res.data) ? res.data : [])
  const container = document.getElementById('burns-table-container')

  if (!records.length) {
    container.innerHTML = '<p style="font-size:12px;color:#5f6368;margin-top:12px;">No pending burn requests.</p>'
    return
  }

  const rows = records.map(r => `
    <tr>
      <td title="${r.id ?? ''}">${truncate(r.id ?? '', 20)}</td>
      <td>${r.vendorId ?? ''}</td>
      <td>${r.amount ?? ''}</td>
      <td>${r.burnRef ?? ''}</td>
      <td><span class="pill pill-${(r.status ?? '').toLowerCase()}">${r.status ?? ''}</span></td>
      <td>${r.initiatedAt ?? ''}</td>
      <td>
        <button class="btn btn-secondary" style="padding:3px 8px;font-size:11px"
          onclick="copyToApproveReject('${r.id ?? ''}')">Use ID</button>
      </td>
    </tr>`).join('')

  container.innerHTML = `
    <table class="data-table">
      <thead><tr>
        <th>TX ID</th><th>Vendor</th><th>Amount</th><th>Form IM Ref</th>
        <th>Status</th><th>Initiated At</th><th></th>
      </tr></thead>
      <tbody>${rows}</tbody>
    </table>`
})

function copyToApproveReject(txId) {
  document.getElementById('c-resolve-id').value = txId
}

function truncate(s, n) {
  return s.length > n ? s.slice(0, n) + '…' : s
}

document.getElementById('btn-c-approve').addEventListener('click', async () => {
  clearStatus('st-c-resolve')
  const id = val('c-resolve-id')
  if (!id) return setStatus('st-c-resolve', 'Burn TX ID is required.', true)
  const res = await apiFetch('POST', `/burns/${encodeURIComponent(id)}/approve`)
  if (!res) return
  res.ok
    ? setStatus('st-c-resolve', `Burn "${truncate(id, 20)}" approved. Tokens destroyed.`, false)
    : setStatus('st-c-resolve', res.data?.error ?? 'Error approving burn.', true)
})

document.getElementById('btn-c-reject').addEventListener('click', async () => {
  clearStatus('st-c-resolve')
  const id = val('c-resolve-id')
  if (!id) return setStatus('st-c-resolve', 'Burn TX ID is required.', true)
  const res = await apiFetch('POST', `/burns/${encodeURIComponent(id)}/reject`)
  if (!res) return
  res.ok
    ? setStatus('st-c-resolve', `Burn "${truncate(id, 20)}" rejected. Tokens returned to vendor.`, false)
    : setStatus('st-c-resolve', res.data?.error ?? 'Error rejecting burn.', true)
})

document.getElementById('btn-c-freeze').addEventListener('click', async () => {
  clearStatus('st-c-freeze')
  const id = val('c-freeze-id')
  if (!id) return setStatus('st-c-freeze', 'Wallet ID is required.', true)
  const res = await apiFetch('POST', `/wallets/${encodeURIComponent(id)}/freeze`)
  if (!res) return
  res.ok
    ? setStatus('st-c-freeze', `Wallet "${id}" frozen.`, false)
    : setStatus('st-c-freeze', res.data?.error ?? 'Error freezing wallet.', true)
})

document.getElementById('btn-c-unfreeze').addEventListener('click', async () => {
  clearStatus('st-c-freeze')
  const id = val('c-freeze-id')
  if (!id) return setStatus('st-c-freeze', 'Wallet ID is required.', true)
  const res = await apiFetch('POST', `/wallets/${encodeURIComponent(id)}/unfreeze`)
  if (!res) return
  res.ok
    ? setStatus('st-c-freeze', `Wallet "${id}" unfrozen.`, false)
    : setStatus('st-c-freeze', res.data?.error ?? 'Error unfreezing wallet.', true)
})

document.getElementById('btn-c-claim').addEventListener('click', async () => {
  clearStatus('st-c-claim')
  const cbosWalletId = val('c-claim-wallet')
  if (!cbosWalletId) return setStatus('st-c-claim', 'CBOS Wallet ID is required.', true)
  const res = await apiFetch('POST', '/fees/claim', { cbosWalletId })
  if (!res) return
  res.ok
    ? setStatus('st-c-claim', 'Fees claimed and credited to CBOS wallet.', false)
    : setStatus('st-c-claim', res.data?.error ?? 'Error claiming fees.', true)
})

document.getElementById('btn-c-aml').addEventListener('click', async () => {
  clearStatus('st-c-aml')
  const threshold = val('c-aml-threshold')
  if (!threshold) return setStatus('st-c-aml', 'Threshold is required.', true)
  const pageSize = val('c-aml-pagesize') || '10'
  const bookmark = val('c-aml-bookmark')
  const params = new URLSearchParams({ threshold, pageSize, bookmark })
  const res = await apiFetch('GET', '/tokens/large-transactions?' + params)
  if (!res) return
  !res.ok && setStatus('st-c-aml', res.data?.error ?? 'Error running AML query.', true)
})

// ── Audit ─────────────────────────────────────────────────

document.getElementById('btn-a-wallet').addEventListener('click', async () => {
  clearStatus('st-a-wallet')
  const id = val('a-wallet-id')
  if (!id) return setStatus('st-a-wallet', 'Wallet ID is required.', true)
  const res = await apiFetch('GET', `/wallets/${encodeURIComponent(id)}`)
  if (res && !res.ok) setStatus('st-a-wallet', res.data?.error ?? 'Not found.', true)
})

document.getElementById('btn-a-balance').addEventListener('click', async () => {
  clearStatus('st-a-wallet')
  const id = val('a-wallet-id')
  if (!id) return setStatus('st-a-wallet', 'Wallet ID is required.', true)
  const res = await apiFetch('GET', `/wallets/${encodeURIComponent(id)}/balance`)
  if (res && !res.ok) setStatus('st-a-wallet', res.data?.error ?? 'Not found.', true)
})

document.getElementById('btn-a-history').addEventListener('click', async () => {
  clearStatus('st-a-history')
  const id = val('a-history-id')
  if (!id) return setStatus('st-a-history', 'Wallet ID is required.', true)
  const res = await apiFetch('GET', `/wallets/${encodeURIComponent(id)}/history`)
  if (res && !res.ok) setStatus('st-a-history', res.data?.error ?? 'Error fetching history.', true)
})

document.getElementById('btn-a-txwallet').addEventListener('click', async () => {
  clearStatus('st-a-txwallet')
  const walletId = val('a-txwallet-id')
  if (!walletId) return setStatus('st-a-txwallet', 'Wallet ID is required.', true)
  const pageSize = val('a-txwallet-pagesize') || '10'
  const bookmark = val('a-txwallet-bookmark')
  const params = new URLSearchParams({ walletId, pageSize, bookmark })
  const res = await apiFetch('GET', '/tokens/transactions?' + params)
  if (res && !res.ok) setStatus('st-a-txwallet', res.data?.error ?? 'Error fetching transactions.', true)
})

document.getElementById('btn-a-supply').addEventListener('click', async () => {
  clearStatus('st-a-supply')
  const res = await apiFetch('GET', '/tokens/supply')
  if (res && !res.ok) setStatus('st-a-supply', res.data?.error ?? 'Error fetching supply.', true)
})

document.getElementById('btn-a-tx').addEventListener('click', async () => {
  clearStatus('st-a-tx')
  const id = val('a-tx-id')
  if (!id) return setStatus('st-a-tx', 'Transaction ID is required.', true)
  const res = await apiFetch('GET', `/transactions/${encodeURIComponent(id)}`)
  if (res && !res.ok) setStatus('st-a-tx', res.data?.error ?? 'Not found.', true)
})

document.getElementById('btn-a-burn').addEventListener('click', async () => {
  clearStatus('st-a-burn')
  const id = val('a-burn-id')
  if (!id) return setStatus('st-a-burn', 'Burn TX ID is required.', true)
  const res = await apiFetch('GET', `/burns/${encodeURIComponent(id)}`)
  if (res && !res.ok) setStatus('st-a-burn', res.data?.error ?? 'Not found.', true)
})

document.getElementById('btn-a-role').addEventListener('click', async () => {
  clearStatus('st-a-role')
  const role = val('a-role-filter')
  const pageSize = val('a-role-pagesize') || '10'
  const bookmark = val('a-role-bookmark')
  const params = new URLSearchParams({ role, pageSize, bookmark })
  const res = await apiFetch('GET', '/wallets?' + params)
  if (res && !res.ok) setStatus('st-a-role', res.data?.error ?? 'Error fetching wallets.', true)
})

// ── Admin ─────────────────────────────────────────────────

document.getElementById('btn-admin-init').addEventListener('click', async () => {
  clearStatus('st-admin-init')
  const res = await apiFetch('POST', '/ledger/init')
  if (!res) return
  res.ok
    ? setStatus('st-admin-init', 'Ledger initialised. CBOS_WALLET created.', false)
    : setStatus('st-admin-init', res.data?.error ?? 'Error initialising ledger.', true)
})

document.getElementById('btn-admin-kyc').addEventListener('click', async () => {
  clearStatus('st-admin-kyc')
  const id = val('admin-kyc-id'), newTier = val('admin-kyc-tier')
  if (!id) return setStatus('st-admin-kyc', 'Wallet ID is required.', true)
  const res = await apiFetch('PUT', `/wallets/${encodeURIComponent(id)}/kyc`, { newTier })
  if (!res) return
  res.ok
    ? setStatus('st-admin-kyc', `KYC tier updated to ${newTier} for "${id}".`, false)
    : setStatus('st-admin-kyc', res.data?.error ?? 'Error updating KYC tier.', true)
})
