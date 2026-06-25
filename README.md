# TSS

CLI and HTTP API wrapper around [tss-lib](https://github.com/binance-chain/tss-lib) for threshold ECDSA key generation, EVM transaction signing, and key resharing.

Each party runs an independent `tss` HTTP server. Parties discover each other over the network, run a distributed ceremony, and store an encrypted secret share locally. No single party ever holds the full private key.

For legacy CLI detail, see [doc/UserGuide.md](doc/UserGuide.md).

## Concepts

| Term | Meaning |
|------|---------|
| **Party** | One participant in the threshold scheme, identified by a unique moniker |
| **Vault** | A named key directory under `.tss/<home>/<vault>/` |
| **t-of-n** | `n` parties hold shares; any `t + 1` parties can sign |
| **Channel** | A session identifier (`channel_id`) used to authenticate peers during bootstrapping |
| **Home** | Per-party data directory name (e.g. `node1` → `.tss/node1/`) |

### Vault layout

After `init`, each vault contains:

```
.tss/<home>/<vault>/
├── config.json   # party config (moniker, listen addr, peers, threshold, …)
└── node_key      # libp2p identity key

# After keygen or regroup:
├── sk.json       # encrypted secret share
└── pk.json       # encrypted public key material
```

## Prerequisites

- Go 1.12+ (see `go.mod`)
- All participating parties must be able to reach each other over the network (LAN, or WAN with relay/bootstrap support)

> **Security note:** Passwords in request bodies or command-line flags are intended for testing only. In production, use secure secret management and TLS.

---

## 1. Build and start the server

```bash
go build
```

The binary starts an HTTP server (default port `8000`):

```bash
# Party A
./tss --port 8001

# Party B
./tss --port 8002

# Party C
./tss --port 8003
```

On macOS, if Gatekeeper blocks the binary:

```bash
xattr -d com.apple.quarantine ./tss
```

### Response format

All API responses use a common envelope:

```json
{
  "success": true,
  "data": { }
}
```

```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "..."
  }
}
```

---

## 2. Initialize parties

Call `POST /init` on every party before keygen. This creates the vault directory, generates a libp2p key pair, and writes `config.json`.

`listen_address` is **required** — pin a fixed port for each party.

```bash
# Party A
curl -s -X POST http://localhost:8001/init \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node1",
    "vault": "alice",
    "moniker": "node1",
    "password": "123456789",
    "listen_address": "/ip4/0.0.0.0/tcp/10000"
  }'

# Party B
curl -s -X POST http://localhost:8002/init \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node2",
    "vault": "alice",
    "moniker": "node2",
    "password": "123456789",
    "listen_address": "/ip4/0.0.0.0/tcp/20000"
  }'

# Party C
curl -s -X POST http://localhost:8003/init \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node3",
    "vault": "alice",
    "moniker": "node3",
    "password": "123456789",
    "listen_address": "/ip4/0.0.0.0/tcp/30000"
  }'
```

Example response:

```json
{
  "success": true,
  "data": {
    "message": "Node has been initialized successfully",
    "id": "12D3KooW...",
    "home": ".tss/node1",
    "vault": "alice",
    "moniker": "node1",
    "listen_address": "/ip4/0.0.0.0/tcp/10000"
  }
}
```

| Field | Description |
|-------|-------------|
| `home` | Party data directory name (stored under `.tss/<home>/`) |
| `vault` | Vault subdirectory name |
| `moniker` | Unique party name (must not contain `@`) |
| `password` | Vault encryption password (min 9 characters) |
| `listen_address` | libp2p listen multiaddr, e.g. `/ip4/0.0.0.0/tcp/10000` |

> Re-initializing an existing vault overwrites `config.json`, `node_key`, `pk.json`, and `sk.json` without confirmation.

---

## 3. Set up a channel

Before keygen or sign, one party generates a **channel id** and shares it with all participants.

```bash
curl -s -X POST http://localhost:8001/channel \
  -H "Content-Type: application/json" \
  -d '{"expire": 30}'
```

Example response:

```json
{
  "success": true,
  "data": {
    "channel_id": "802671B1B19"
  }
}
```

- The channel id is 11 characters: a 3-digit random prefix + a hex-encoded expiry timestamp.
- `expire` sets lifetime in minutes.
- All parties in the same session must use the **same** `channel_id`.
- For HTTP keygen and sign, the vault `password` is also used as the channel password.

---

## 4. Generate a key (keygen)

All `n` parties must call `POST /keygen` concurrently with matching parameters. Each party ends up with an encrypted share of the same ECDSA key.

On a LAN, parties discover each other automatically via SSDP. Start all requests within a short window:

```bash
# All 3 parties — run concurrently
curl -s -X POST http://localhost:8001/keygen \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node1",
    "vault": "alice",
    "password": "123456789",
    "parties": 3,
    "threshold": 1,
    "channel_id": "802671B1B19"
  }'

curl -s -X POST http://localhost:8002/keygen \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node2",
    "vault": "alice",
    "password": "123456789",
    "parties": 3,
    "threshold": 1,
    "channel_id": "802671B1B19"
  }'

curl -s -X POST http://localhost:8003/keygen \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node3",
    "vault": "alice",
    "password": "123456789",
    "parties": 3,
    "threshold": 1,
    "channel_id": "802671B1B19"
  }'
```

Example response:

```json
{
  "success": true,
  "data": {
    "address": "0x...",
    "pubkey": "0x04..."
  }
}
```

If `sk.json` already exists, keygen returns the existing address and pubkey without re-running the ceremony.

### What happens during keygen

1. **SSDP discovery** — parties find each other's listen addresses on the LAN.
2. **Raw TCP bootstrapping** — parties exchange libp2p IDs and addresses, encrypted with channel credentials.
3. **libp2p session** — parties run the tss-lib keygen protocol.
4. **Persist** — each party writes `sk.json`, `pk.json`, and updates `config.json`.

---

## 5. Sign a transaction

Signing requires at least `t + 1` parties from the original `n`. For `parties=3`, `threshold=1`, any 2 parties suffice.

`POST /sign` builds an EIP-1559 ETH transfer, runs the threshold signing protocol, and returns the signed raw transaction.

### Generate a new channel

```bash
curl -s -X POST http://localhost:8001/channel \
  -H "Content-Type: application/json" \
  -d '{"expire": 30}'
```

### Run sign on t+1 parties

```bash
# Party A
curl -s -X POST http://localhost:8001/sign \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node1",
    "vault": "alice",
    "password": "123456789",
    "channel_id": "5185D3EF597",
    "rpc_url": "http://localhost:8545",
    "to_address": "0xRecipientAddress...",
    "amount": "1"
  }'

# Party B
curl -s -X POST http://localhost:8002/sign \
  -H "Content-Type: application/json" \
  -d '{
    "home": "node2",
    "vault": "alice",
    "password": "123456789",
    "channel_id": "5185D3EF597",
    "rpc_url": "http://localhost:8545",
    "to_address": "0xRecipientAddress...",
    "amount": "1"
  }'
```

Example response:

```json
{
  "success": true,
  "data": {
    "raw_tx": "0x02f8..."
  }
}
```

| Field | Description |
|-------|-------------|
| `rpc_url` | EVM JSON-RPC endpoint |
| `to_address` | Recipient address |
| `amount` | Transfer amount in ETH |

The server fetches chain ID and nonce from `rpc_url`, constructs an EIP-1559 transfer, and signs its hash.

> **Note:** Sign uses libp2p only (no SSDP). Parties must already know each other's peer info from the prior keygen session stored in `config.json`.

---

## 6. Reshare a key (regroup)

`regroup` rotates secret shares while keeping the same public key (EVM address). Use it to refresh shares, replace a party, or change the `t`-`n` policy.

> **Note:** Regroup is currently available via the CLI only (no HTTP endpoint yet). To use CLI commands, switch `main.go` back to `cmd.Execute()` and rebuild.

At least `old_threshold + 1` old-committee parties must participate.

### Roles

| Role | Flags | Description |
|------|-------|-------------|
| Old + new committee | `--is_old true --is_new_member true` | Holds an existing share; contributes to resharing and receives a new share |
| New committee only | `--is_old false --is_new_member true` | Brand-new party; receives a share (set automatically if no `sk.json` exists) |

### Example: refresh all 3 parties (1-of-3 → 1-of-3)

```bash
# Generate a new channel
./tss channel --channel_expire 30

# Party A & B — old committee, also new committee
./tss regroup --home .tss/node1 --vault_name "alice" \
  --password "123456789" --parties 3 --threshold 1 \
  --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "3415D3FBE00" \
  --is_old true --is_new_member true \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/10001"

./tss regroup --home .tss/node2 --vault_name "alice" \
  --password "123456789" --parties 3 --threshold 1 \
  --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "3415D3FBE00" \
  --is_old true --is_new_member true \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/20001"

# Party C — re-init then join as new committee only
./tss init --home .tss/node3 --vault_name "alice" \
  --moniker "node3" --password "123456789" \
  --p2p.listen "/ip4/0.0.0.0/tcp/30000"

./tss regroup --home .tss/node3 --vault_name "alice" \
  --password "123456789" --parties 3 --threshold 1 \
  --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "3415D3FBE00" \
  --is_old false --is_new_member true
```

On success, each participating party's `sk.json`, `pk.json`, and `config.json` are updated. The EVM address remains unchanged.

---

## End-to-end workflow (HTTP, localhost)

```bash
# 0. Build
go build -o tss

# 1. Start 3 servers (separate terminals)
./tss --port 8001
./tss --port 8002
./tss --port 8003

# 2. Init 3 parties
curl -X POST http://localhost:8001/init -H "Content-Type: application/json" \
  -d '{"home":"node1","vault":"alice","moniker":"node1","password":"123456789","listen_address":"/ip4/0.0.0.0/tcp/10000"}'
curl -X POST http://localhost:8002/init -H "Content-Type: application/json" \
  -d '{"home":"node2","vault":"alice","moniker":"node2","password":"123456789","listen_address":"/ip4/0.0.0.0/tcp/20000"}'
curl -X POST http://localhost:8003/init -H "Content-Type: application/json" \
  -d '{"home":"node3","vault":"alice","moniker":"node3","password":"123456789","listen_address":"/ip4/0.0.0.0/tcp/30000"}'

# 3. Channel
curl -X POST http://localhost:8001/channel -H "Content-Type: application/json" -d '{"expire":30}'
# → use the returned channel_id below

# 4. Keygen (all 3 parties, concurrently)
curl -X POST http://localhost:8001/keygen -H "Content-Type: application/json" \
  -d '{"home":"node1","vault":"alice","password":"123456789","parties":3,"threshold":1,"channel_id":"<CHANNEL_ID>"}'
curl -X POST http://localhost:8002/keygen -H "Content-Type: application/json" \
  -d '{"home":"node2","vault":"alice","password":"123456789","parties":3,"threshold":1,"channel_id":"<CHANNEL_ID>"}'
curl -X POST http://localhost:8003/keygen -H "Content-Type: application/json" \
  -d '{"home":"node3","vault":"alice","password":"123456789","parties":3,"threshold":1,"channel_id":"<CHANNEL_ID>"}'

# 5. Sign (2 of 3 parties)
curl -X POST http://localhost:8001/channel -H "Content-Type: application/json" -d '{"expire":30}'
curl -X POST http://localhost:8001/sign -H "Content-Type: application/json" \
  -d '{"home":"node1","vault":"alice","password":"123456789","channel_id":"<CHANNEL_ID>","rpc_url":"http://localhost:8545","to_address":"0x...","amount":"1"}'
curl -X POST http://localhost:8002/sign -H "Content-Type: application/json" \
  -d '{"home":"node2","vault":"alice","password":"123456789","channel_id":"<CHANNEL_ID>","rpc_url":"http://localhost:8545","to_address":"0x...","amount":"1"}'
```

---

## HTTP API reference

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/channel` | Generate a channel id |
| `POST` | `/init` | Initialize a party vault |
| `POST` | `/keygen` | Distributed key generation (all `n` parties) |
| `POST` | `/sign` | Threshold-sign an EVM transfer (`t+1` parties) |

---

## CLI reference (optional)

The CLI commands (`init`, `channel`, `keygen`, `sign`, `regroup`, `describe`) remain in `cmd/` but are not the default entry point. To use them, restore `cmd.Execute()` in `main.go` and rebuild.

```
tss init       Create vault, generate p2p key pair
tss channel    Generate a channel id for a session
tss keygen     Distributed key generation (all n parties)
tss sign       Threshold-sign an EVM transaction (t+1 parties)
tss regroup    Reshare / rotate keys (old_t+1 old parties)
tss describe   Show vault config and address
```

```bash
./tss --help
./tss <command> --help
```

---

## Network bootstrapping

Each ceremony uses up to three discovery layers:

| Layer | Used by | Description |
|-------|---------|-------------|
| SSDP | keygen, regroup | LAN broadcast to discover peer listen addresses (unencrypted) |
| Raw TCP | keygen, regroup | Encrypted peer handshake using channel id + password |
| libp2p | keygen, sign, regroup | Formal P2P transport for the tss-lib protocol |

Sign relies on libp2p only (peer info from `config.json`), so it can work across WAN when bootstrap/relay servers are configured.

### NAT traversal

libp2p supports three NAT traversal mechanisms:

1. **UPnP / NAT-PMP** — automatic port forwarding (best option when the router supports it)
2. **STUN / hole-punching** — peer-routing and external address discovery
3. **p2p-circuit (relay)** — proxy through relay nodes; required for symmetric NAT clients

| Role | Full cone | Restricted cone | Port-restricted | Symmetric NAT |
|------|-----------|-----------------|-------------------|---------------|
| Bootstrap server | ✓ | ✘ | ✘ | ✘ |
| Relay server | ✓ | ✘ | ✘ | ✘ |
| Client | ✓ | ✓ | ✓ | ✓ (relay needed) |

On a LAN, parties connect directly without bootstrap or relay servers.

---

## Security guidelines

- Use `n > t + 1` so a lost party can be recovered via regroup without full quorum.
- Validate keygen by signing test transactions with different party subsets before depositing funds.
- Regroup shares periodically; destroy old backups after a successful regroup.
- Agree on a new channel id for every session (keygen, sign, regroup).
- Never run two ceremonies with the same channel id on the same vault simultaneously.
- Store encrypted `sk.json` backups in secure, offline locations.
- Do not expose the HTTP server to untrusted networks without authentication and TLS.
