# TSS

CLI and P2P transport wrapper around [tss-lib](https://github.com/binance-chain/tss-lib) for threshold ECDSA key generation, signing, and resharing.

Each party runs an independent `tss` process. Parties discover each other over the network, run a distributed ceremony, and store an encrypted secret share locally. No single party ever holds the full private key.

For command-level detail, see [doc/UserGuide.md](doc/UserGuide.md).

## Concepts

| Term | Meaning |
|------|---------|
| **Party** | One participant in the threshold scheme, identified by a unique moniker |
| **Vault** | A named key directory under `--home` (default `~/.tss/<vault_name>/`) |
| **t-of-n** | `n` parties hold shares; any `t + 1` parties can sign |
| **Channel** | A session identifier (`channel_id`) plus a shared password (`channel_password`) used to authenticate peers during bootstrapping |

### Vault layout

After `init`, each vault contains:

```
.tss/<vault_name>/
├── config.json   # party config (moniker, listen addr, peers, threshold, …)
└── node_key      # libp2p identity key

# After keygen or regroup:
├── sk.json       # encrypted secret share
└── pk.json       # encrypted public key material
```

## Prerequisites

- Go 1.12+ (see `go.mod`)
- All participating parties must be able to reach each other over the network (LAN, or WAN with relay/bootstrap support)

> **Security note:** `--password` and `--channel_password` on the command line are intended for testing only. In production, omit them and enter passwords interactively.

---

## 1. Build the binary

```bash
git clone https://github.com/binance-chain/tss
cd tss
go build
```

On macOS, if Gatekeeper blocks the binary:

```bash
xattr -d com.apple.quarantine ./tss
```

---

## 2. Initialize parties

Run `tss init` once on every machine that will participate. This creates the vault directory, generates a libp2p key pair, and writes `config.json`.

```bash
# Party A
./tss init --home .tss/node1 --vault_name "alice" --moniker "node1" --password "123456789" --p2p.listen "/ip4/0.0.0.0/tcp/10000" 

# Party B
./tss init --home .tss/node2 --vault_name "alice" --moniker "node2" --password "123456789" --p2p.listen "/ip4/0.0.0.0/tcp/20000" 

# Party C
./tss init --home .tss/node3 --vault_name "alice" --moniker "node3" --password "123456789" --p2p.listen "/ip4/0.0.0.0/tcp/30000" 
```

| Flag | Description |
|------|-------------|
| `--home` | Root directory for vault data (default `~/.tss`) |
| `--vault_name` | Vault subdirectory name |
| `--moniker` | Human-readable party name (must be unique, must not contain `@`) |
| `--password` | Vault encryption password (min 9 characters) |
| `--p2p.listen` | Optional fixed listen multiaddr, e.g. `/ip4/0.0.0.0/tcp/55101`. If omitted, a random port on `0.0.0.0` is chosen |

Verify a vault with:

```bash
./tss describe --home .tss/node1 --vault_name "alice" --password "123456789"
```

---

## 3. Set up a channel

Before keygen, sign, or regroup, the coordinator generates a **channel id** and agrees on a **channel password** with all participants out of band (e.g. in person or over a secure channel).

```bash
./tss channel --channel_expire 30
```

Example output:

```
channel id: 802671B1B19
```

- The channel id is 11 characters: a 3-digit random prefix + a hex-encoded expiry timestamp.
- `--channel_expire` sets lifetime in minutes (default 30).
- All parties in the same session must use the **same** `channel_id` and `channel_password`.

---

## 4. Generate a key (keygen)

All `n` parties must run `keygen` concurrently with matching parameters. Each party ends up with an encrypted share of the same ECDSA key. The derived EVM address is logged on completion.

### LAN / localhost (SSDP peer discovery)

On a local network, parties discover each other automatically via SSDP. Start all parties within a short window:

```bash
# All 3 parties — run concurrently
./tss keygen --home .tss/node1 --vault_name "alice" \
  --parties 3 --threshold 1 \
  --password "123456789" \
  --channel_password "123456789" \
  --channel_id "802671B1B19"

./tss keygen --home .tss/node2 --vault_name "alice" \
  --parties 3 --threshold 1 \
  --password "123456789" \
  --channel_password "123456789" \
  --channel_id "802671B1B19"

./tss keygen --home .tss/node3 --vault_name "alice" \
  --parties 3 --threshold 1 \
  --password "123456789" \
  --channel_password "123456789" \
  --channel_id "802671B1B19"
```

### VPC / no SSDP (explicit peer addresses)

In environments without SSDP (e.g. AWS VPC), pin listen addresses at `init` and pass every other party's address at `keygen`:

```bash
# Init with fixed ports
./tss init --home .tss/node1 --vault_name "alice" --moniker "node1" --password 123456789 \
  --p2p.listen "/ip4/0.0.0.0/tcp/10000"
./tss init --home .tss/node2 --vault_name "alice" --moniker "node2" --password 123456789 \
  --p2p.listen "/ip4/0.0.0.0/tcp/20000"
./tss init --home .tss/node3 --vault_name "alice" --moniker "node3" --password 123456789 \
  --p2p.listen "/ip4/0.0.0.0/tcp/30000"

# Keygen — each party lists the other parties' listen addrs
./tss keygen --home .tss/node1 --vault_name "alice" --parties 3 --threshold 1 \
  --password "123456789" --channel_password "123456789" --channel_id "20963C1108C" \
  --p2p.peer_addrs "/ip4/0.0.0.0/tcp/20000","/ip4/0.0.0.0/tcp/30000"

./tss keygen --home .tss/node2 --vault_name "alice" --parties 3 --threshold 1 \
  --password "123456789" --channel_password "123456789" --channel_id "20963C1108C" \
  --p2p.peer_addrs "/ip4/0.0.0.0/tcp/10000","/ip4/0.0.0.0/tcp/30000"

./tss keygen --home .tss/node3 --vault_name "alice" --parties 3 --threshold 1 \
  --password "123456789" --channel_password "123456789" --channel_id "20963C1108C" \
  --p2p.peer_addrs "/ip4/0.0.0.0/tcp/10000","/ip4/0.0.0.0/tcp/20000"
```

### What happens during keygen

1. **Raw TCP bootstrapping** — parties exchange libp2p IDs and listen addresses, encrypted with the channel credentials.
2. **libp2p session** — parties connect and run the tss-lib keygen protocol.
3. **Persist** — each party writes `sk.json` and `pk.json` and updates `config.json` with peer info.

On success, all parties log the same EVM address:

```
INFO  tss: [party1] evm address is: 0x...
```

---

## 5. Sign a message

Signing requires at least `t + 1` parties from the original `n`. For a `3-of-2` scheme (`parties=3`, `threshold=1`), any 2 parties suffice.

The `sign` command builds an EVM transfer transaction, hashes it, and runs the threshold signing protocol. The signed raw transaction is logged as hex.

### Prepare a new channel

Generate a fresh channel for each signing session:

```bash
./tss channel --channel_expire 30
# share channel id and channel password with signing parties
```

### Run sign on t+1 parties

```bash
# Party A
./tss sign --home .tss/node1 --vault_name "alice" \
  --password "123456789" \
  --channel_password "123456789" \
  --channel_id "5185D3EF597" \
  --rpc_url "http://localhost:8545" \
  --to_address "0xRecipientAddress..." \
  --amount "1"

# Party B (only t+1 parties needed; Party C is not required)
./tss sign --home .tss/node2 --vault_name "alice" \
  --password "123456789" \
  --channel_password "123456789" \
  --channel_id "5185D3EF597" \
  --rpc_url "http://localhost:8545" \
  --to_address "0xRecipientAddress..." \
  --amount "1"
```

| Flag | Description |
|------|-------------|
| `--rpc_url` | EVM JSON-RPC endpoint (default `http://localhost:8545`) |
| `--to_address` | Recipient address |
| `--amount` | Transfer amount in ETH (default `1`) |

The CLI fetches chain ID and nonce from `--rpc_url`, constructs an EIP-1559 transfer, and signs its hash. On completion:

```
INFO  tss: [party1] signed tx: <hex-encoded raw transaction>
```

> **Note:** Sign uses libp2p only (no SSDP). Parties must already know each other's peer info from the prior keygen session stored in `config.json`, or be reachable via bootstrap/relay in WAN deployments.

---

## 6. Reshare a key (regroup)

`regroup` rotates secret shares while keeping the same public key (EVM address). Use it to:

- Periodically refresh shares (recommended monthly)
- Replace a compromised or lost party
- Change the `t`-`n` policy (e.g. 1-of-3 → 2-of-4)

At least `old_threshold + 1` old-committee parties must participate in the resharing ceremony.

### Roles

| Role | Flags | Description |
|------|-------|-------------|
| Old + new committee | `--is_old true --is_new_member true` | Holds an existing share; contributes to resharing and receives a new share |
| New committee only | `--is_old false --is_new_member true` | Brand-new party; receives a share (set automatically if no `sk.json` exists) |
| Skipped | `--is_old false --is_new_member true` | Existing party that does not contribute old shares but still receives a new share |

### Example: refresh all 3 parties (1-of-3 → 1-of-3)

Generate a new channel, then run regroup on all parties. Two old parties participate in signing; the third only joins the new committee:

```bash
# New channel
./tss channel --channel_expire 30
# channel id: 3415D3FBE00

# Party A & B — old committee, also new committee
./tss regroup --home .tss/node1 --vault_name "alice" \
  --password "123456789" --parties 3 --threshold 1 \
  --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "3415D3FBE00" \
  --is_old true --is_new_member true \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/10001"

./tss regroup --home .tss/node3 --vault_name "alice" \
  --password "123456789" --parties 3 --threshold 1 \
  --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "3415D3FBE00" \
  --is_old true --is_new_member true
  --p2p.new_listen "/ip4/0.0.0.0/tcp/20001"

# Party C — new committee only (does not contribute old shares)
# Before regroup party C as new party, need to re-init party C
./tss init --home .tss/node3 --vault_name "alice" \
  --moniker "node3" --password 123456789 \
  --p2p.listen "/ip4/0.0.0.0/tcp/30000"
# Then run regroup with party C
./tss regroup --home .tss/node3 --vault_name "alice" \
  --password "123456789" --parties 3 --threshold 1 \
  --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "3415D3FBE00" \
  --is_old false --is_new_member true
```

### Example: replace parties in a VPC (no SSDP)

When adding a new party D to replace C:

```bash
# 1. Init the new party
./tss init --home .tss/node4 --vault_name "alice" \
  --moniker "node4" --password "123456789" \
  --p2p.listen "/ip4/0.0.0.0/tcp/40000"

# 2. Regroup — old parties A & B (old + new committee)
./tss regroup --is_old true --is_new_member true \
  --home .tss/node1 --vault_name "alice" --password "123456789" \
  --parties 3 --threshold 1 --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "20963C1108C" \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/10001" \
  --p2p.new_peer_addrs "/ip4/0.0.0.0/tcp/10000","/ip4/0.0.0.0/tcp/20000","/ip4/0.0.0.0/tcp/20001","/ip4/0.0.0.0/tcp/40000"

./tss regroup --is_old true --is_new_member true \
  --home .tss/node2 --vault_name "alice" --password "123456789" \
  --parties 3 --threshold 1 --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "20963C1108C" \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/20001" \
  --p2p.new_peer_addrs "/ip4/0.0.0.0/tcp/10000","/ip4/0.0.0.0/tcp/10001","/ip4/0.0.0.0/tcp/20000","/ip4/0.0.0.0/tcp/40000"

# 3. New party D — new committee only
./tss regroup --is_old false --is_new_member true \
  --home .tss/node4 --vault_name "alice" --password "123456789" \
  --parties 3 --threshold 1 --new_parties 3 --new_threshold 1 \
  --channel_password "123456789" --channel_id "20963C1108C" \
  --p2p.new_peer_addrs "/ip4/0.0.0.0/tcp/10000","/ip4/0.0.0.0/tcp/10001","/ip4/0.0.0.0/tcp/20000","/ip4/0.0.0.0/tcp/20001"
```

On success, each participating party's `sk.json`, `pk.json`, and `config.json` are updated in place. The EVM address remains unchanged.

After regroup, securely destroy old share backups.

---

## End-to-end workflow (localhost)

```bash
# 0. Build
go build -o tss

# 1. Init 3 parties
./tss init --home .tss/node1 --vault_name "alice" \
  --moniker "node1" --password "123456789" \ 
  --p2p.new_listen "/ip4/0.0.0.0/tcp/10000"

./tss init --home .tss/node2 --vault_name "alice" \
  --moniker "node2" --password "123456789" \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/20000"

./tss init --home .tss/node3 --vault_name "alice" \
  --moniker "node3" --password "123456789" \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/30000"

# 2. Channel
./tss channel --channel_expire 30
# → use the printed channel id below

# 3. Keygen (all 3 parties, concurrently)
./tss keygen --home .tss/node1 --vault_name "alice" \
  --parties 3 --threshold 1 --password "123456789" \
  --channel_password "123456789" --channel_id "<CHANNEL_ID>"

./tss keygen --home .tss/node2 --vault_name "alice" \
  --parties 3 --threshold 1 --password "123456789" \
  --channel_password "123456789" --channel_id "<CHANNEL_ID>"

./tss keygen --home .tss/node3 --vault_name "alice" \
  --parties 3 --threshold 1 --password "123456789" \
  --channel_password "123456789" --channel_id "<CHANNEL_ID>"

# 4. Sign (2 of 3 parties)
./tss sign --home .tss/node1 --vault_name "alice" \
  --password "123456789" --channel_password "123456789" --channel_id "<CHANNEL_ID>" \
  --rpc_url http://localhost:8545 --to_address "0x..." --amount "1"

./tss sign --home .tss/node2 --vault_name "alice" \
  --password "123456789" --channel_password "123456789" --channel_id "<CHANNEL_ID>" \
  --rpc_url http://localhost:8545 --to_address "0x..." --amount "1"

# 5. Regroup (refresh shares)
./tss regroup --home .tss/node1 --vault_name "alice" --password "123456789" \
  --new_parties 3 --new_threshold 1 --parties 3 --threshold 1 \
  --channel_password "123456789" --channel_id "<CHANNEL_ID>" \
  --is_old true --is_new_member true \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/10001" 

./tss regroup --home .tss/node2 --vault_name "alice" --password "123456789" \
  --new_parties 3 --new_threshold 1 --parties 3 --threshold 1 \
  --channel_password "123456789" --channel_id "<CHANNEL_ID>" \
  --is_old true --is_new_member true \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/20001" 

./tss init --home .tss/node4 --vault_name "alice" \
  --moniker "node4" --password "123456789" \
  --p2p.new_listen "/ip4/0.0.0.0/tcp/40000"
./tss regroup --home .tss/node4 --vault_name "alice" --password "123456789" \
  --new_parties 3 --new_threshold 1 --parties 3 --threshold 1 \
  --channel_password "123456789" --channel_id "<CHANNEL_ID>" \
  --is_old false --is_new_member true
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

## CLI reference

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

## Security guidelines

- Use `n > t + 1` so a lost party can be recovered via regroup without full quorum.
- Validate keygen by signing test transactions with different party subsets before depositing funds.
- Regroup shares periodically; destroy old backups after a successful regroup.
- Agree on a new channel password for every session (keygen, sign, regroup).
- Never run two `tss` processes with the same channel id on the same vault simultaneously.
- Store encrypted `sk.json` backups in secure, offline locations.
