# Hawal Digital — User Manual

> **Prototype notice:** This is a thesis demonstration frontend. There is no real authentication. Account numbers are saved in your browser's localStorage and persist between sessions.

---

## Prerequisites

Before using the frontend, two services must be running:

| Service | Command | Default port |
|---|---|---|
| Hyperledger Fabric network | (already deployed) | — |
| REST API | `cd api && go run .` | 8080 |
| Frontend | `cd frontend-demo && python3 -m http.server 3000` | 3000 |

Open **http://localhost:3000** in your browser.

---

## First-Time Setup (Admin)

Before any user flow works, the ledger and test accounts must be initialised. This is a one-time step.

1. From the landing page, click **Admin Setup** at the bottom (or go to `admin.html?admin=true`).
2. Run the steps below in order:

### Step 1 — Init Ledger
Click **Init Ledger (Org1)**. This creates the `CBOS_WALLET` system account. Run exactly once.

### Step 2 — Register Accounts

Use **Register Wallet** to create accounts for each role. Suggested test accounts:

| Account ID | Role | KYC Tier | Org |
|---|---|---|---|
| `diaspora-001` | DIASPORA | 2 | Org1 |
| `relative-001` | RELATIVE | 2 | Org2 |
| `vendor-001` | VENDOR | 2 | Org2 |

- Select **Org1** for DIASPORA accounts.
- Select **Org2** for RELATIVE and VENDOR accounts.

### Step 3 — Fund the Sender
Use **Mint Tokens** to add funds to the sender account:

| Field | Value |
|---|---|
| Wallet ID | `diaspora-001` |
| Amount | `5000` |
| Deposit Reference | `DEP-2024-001` (any unique string) |

> Deposit references are stored permanently on-chain and cannot be reused.

### KYC Tier Limits

| Tier | Per-transaction limit |
|---|---|
| 1 | £1,000 |
| 2 | £10,000 |
| 3 | Unlimited |

Use **Update KYC Tier** to change a tier at any time.

---

## Landing Page

Open `http://localhost:3000`. Four role cards are shown — click the one that matches your role.

| Card | Page | Role |
|---|---|---|
| Sending money abroad | `sender.html` | Diaspora sender (UK) |
| Received money from family | `recipient.html` | Recipient in Sudan |
| Running a local business | `vendor.html` | Merchant/vendor |
| Central Bank of Sudan | `cbos.html` | CBOS regulator |

---

## Sender (Diaspora — UK)

**Account:** Org1 · DIASPORA role

### First visit
Enter your account number (e.g. `diaspora-001`) and click **Get Started**. It is saved for future visits.

### Send Calculator (Home tab)
The calculator on the home screen gives a live fee preview before opening the send flow:

1. Type an amount in the **How much do you want to send?** field.
2. The fee (0.10%, minimum £1) and the exact amount the recipient will receive update instantly.
3. Click **Send Now** to open the send wizard pre-filled with that amount.

### Send Wizard (3 steps)

**Step 1 — Amount**
- Enter the amount in GBP.
- A fee breakdown appears below the field showing: *You send → Fee → Recipient gets*.

**Step 2 — Recipient**
- Enter the recipient's Hawal Digital account number (e.g. `relative-001`).

**Step 3 — Review & Confirm**
- Final summary shows the full breakdown.
- Click **Confirm & Send** to submit the transfer.
- On success, the balance and recent activity update automatically.

### Transfer History tab
Full paginated history of all transfers. Click **Load more** to fetch older records.

---

## Recipient (Sudan)

**Account:** Org2 · RELATIVE role

### First visit
Enter your account number (e.g. `relative-001`) and click **Get Started**.

### Home tab
- Balance is shown in GBP.
- **Recent activity** shows the last 5 transactions as a card list.
- Click **Pay a Merchant** to open the payment flow.

### Pay a Merchant (2 steps)

**Step 1 — Details**
- Enter the merchant's account number (e.g. `vendor-001`).
- Enter the amount.

**Step 2 — Confirm**
- Review the full breakdown (amount + fee).
- Click **Make Payment** to submit.

### Transaction History tab
Full paginated history. Click **Load more** for older records.

---

## Vendor / Merchant (Sudan)

**Account:** Org2 · VENDOR role

### First visit
Enter your account number (e.g. `vendor-001`) and click **Get Started**.

### Home tab
- Balance shows funds received from customers.
- Click **Request Settlement** to convert digital balance to hard currency via the Central Bank.

### Request Settlement (2 steps)

**Step 1 — Details**
- **Invoice Reference** — your import invoice number (e.g. `INV-2024-001`). This is stored on-chain.
- **Amount** — the amount to settle.
- Note: funds are **held** (locked) while the request is under CBOS review.

**Step 2 — Confirm**
- Review the request details.
- Click **Submit Request**. A success toast confirms submission.

### Settlements tab
Shows all settlement requests submitted **in this session** with their current status:

| Status | Meaning |
|---|---|
| Processing | Awaiting CBOS decision |
| Approved | CBOS approved — funds released |
| Declined | CBOS declined — funds returned |

> Requests from previous sessions can be viewed in the Transaction History tab.

### Transaction History tab
Full paginated transaction history.

---

## CBOS Regulator

**Account:** Org3 · Regulatory access (no personal account number needed)

The CBOS dashboard uses a dark sidebar layout with five sections.

### Overview
Real-time statistics read directly from the ledger:

| Stat | Description |
|---|---|
| Total Circulation | Total GBP currently in active circulation |
| Pending Settlements | Number of merchant settlement requests awaiting decision |
| Regulatory Fee Balance | Accumulated fees in the CBOS fee pool |

Click **Refresh** to reload all stats. The timestamp shows when data was last fetched.

Quick-links to Settlement Queue and Fee Collection are shown below the stats.

### Settlement Queue
Lists all pending merchant settlement requests. For each request:

- **Approve** — releases hard currency to the merchant and retires the corresponding tokens from circulation. **Irreversible.**
- **Decline** — returns the held funds to the merchant's account in full.

Both actions show a confirmation modal before executing.

Click **Reload** to refresh the queue.

### AML Monitoring
Search for all transactions above a threshold amount for anti-money laundering review.

1. Enter a minimum amount (GBP).
2. Click **Search**.
3. Results show TX ID, type, sender, recipient, amount, and date.

### Account Controls
Look up any account on the network:

1. Enter an account number and click **Look Up**.
2. View role, balance, verification level, and status.
3. Take regulatory action:
   - **Restrict Account** — freezes the account; all transfers blocked until lifted.
   - **Lift Restriction** — restores the account to active status.

### Fee Collection
Collects all accumulated transaction fees (0.10% per transfer/payment) into the CBOS regulatory account.

- Current fee balance is displayed.
- Click **Collect All Fees** to sweep all fee entries atomically into `CBOS_WALLET`.

---

## Fee Structure

All transfers and payments carry a flat fee of **0.10%** (10 basis points), with a minimum of **£1**.

```
fee = max(1, ceil(amount × 10 / 10000))
```

Examples:

| Amount | Fee | Recipient receives |
|---|---|---|
| £100 | £1 | £99 |
| £500 | £1 | £499 |
| £1,000 | £1 | £999 |
| £5,000 | £5 | £4,995 |
| £10,000 | £10 | £9,990 |

Fees accumulate on-chain and are collected in bulk by CBOS via the Fee Collection panel.

---

## End-to-End Demo Flow

A complete demonstration of all roles in sequence:

1. **Admin** — Init ledger, register `diaspora-001` / `relative-001` / `vendor-001`, mint £5,000 to `diaspora-001`.
2. **Sender** (`diaspora-001`) — Send £500 to `relative-001`. Observe fee deducted.
3. **Recipient** (`relative-001`) — Confirm balance increased by £499. Pay £200 to `vendor-001`.
4. **Vendor** (`vendor-001`) — Confirm £199 received. Submit a settlement request for £150, Invoice Ref `INV-2024-001`.
5. **CBOS** — Open Settlement Queue, approve the £150 request. Check Total Circulation decreased. Collect fees.
6. **CBOS / AML** — Search for transactions above £100 to see the full audit trail.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| "Cannot reach the API" toast | API not running | Run `cd api && go run .` |
| Balance shows `—` after setup | Wrong account number or account not registered | Check Admin → Register Wallet |
| "Account not found" error | Account ID not registered | Register it via Admin Setup |
| "This amount exceeds your verification limit" | Amount above KYC tier limit | Increase tier via Admin → Update KYC Tier |
| Send wizard submits but balance doesn't change | Network delay | Refresh the page |
| Settlement status stays "Processing" | Waiting for CBOS action | Switch to CBOS dashboard and approve |
| Settlements tab shows nothing | Requests submitted in a previous session | Check Transaction History instead |
