# CrossChain UTXO Emergency Restriction: Release Readiness and Activation Runbook

**Status:** Draft — do not merge or deploy without complete sign-off.

**Policy pull request:** #2275

**Mainnet activation heights:**

- `E = 2,256,110`
- `H = 2,256,724`

All timestamps and evidence records in this runbook use UTC. The release
commander must record the final reviewed source revision, signed tag, artifact
checksums, and signatures in the sign-off record before authorizing rollout.

## Purpose and scope

This release introduces a two-stage, mainnet-only consensus restriction for
spends of CrossChain-prefixed (X-address) UTXOs. It restores the legacy dynamic
CrossChain witness behavior used by the live baseline before activation,
temporarily rejects all X-UTXO spending at `E`, and later permits only existing
authorized bridge withdrawal and failed-deposit-refund paths at `H`.

This is a preventive spending restriction. It does not pause inbound ESC
deposits, reverse confirmed transactions, or automatically recover or refund
assets. ESC and Arbiter bridge-process quarantine is a separate operational
control.

## Consensus policy

| Block range | Consensus rule | Operational consequence |
| --- | --- | --- |
| Before `E` | Legacy CrossChain validation remains in force. | Historical replay must remain compatible. |
| `E` through `H - 1` | Reject every transaction that spends an X UTXO. | Type-7 withdrawals and Type-81 refunds are intentionally paused. |
| `H` and later | Allow valid Type-7 V0/V1/V2 withdrawals under their existing authorization, and Type-81/V0 refunds only when every input is an X UTXO and the witness is authorized by the current CrossChain arbiters. | Generic transfers and every other X-UTXO spend remain prohibited. |

The activation heights are compiled into mainnet policy and cannot be changed
through local configuration or command-line flags. A block height is not a
wall-clock guarantee. `H - E` is 614 blocks; do not describe the interval as an
exact duration.

## Critical boundaries and non-claims

- The policy applies to spending an existing X UTXO. It does not prevent the
  creation of a new `TransferCrossChainAsset` deposit.
- The policy does not stop ESC SPV observation, deposit persistence, or recharge
  processing. Those processes require their own verified quarantine controls.
- The policy does not roll back history, recover funds, or create refunds.
- A node that has not upgraded can accept a block that an upgraded node rejects
  at or after `E`. Treat rollout with hard-fork-level operational discipline.

## Mandatory no-go gates

Every item below must be evidenced. Any unchecked item is a no-go decision.

- [ ] Final source revision, signed tag, Linux artifact SHA-256 values, and
      signature verification are recorded.
- [ ] Both CI workflows are green for the exact final source revision.
- [ ] Operations reconfirms that no validator, template service, standby, or
      failover service has deployed the behavior introduced by PR #2273.
- [ ] Candidate-versus-legacy replay/reindex matches through `E - 1` from a
      trusted, immutable chain-data snapshot.
- [ ] Recorded production-format Arbiter Type-7/V1 and Type-81/V0 transactions
      complete validation in the release test environment.
- [ ] ESC inbound deposit/recharge processing and Arbiter deposit, withdrawal,
      refund, proposal, and broadcast workers have separately evidenced
      quarantine.
- [ ] Every active, current-and-next, and standby BPoS operator attests the
      exact signed artifact.
- [ ] Every ELA `createauxblock` / `submitauxblock`, merged-mining, and
      failover template service attests the exact signed artifact.
- [ ] The release commander records a written GO decision with evidence links.

## Artifact production and verification

1. Build the final tagged source in a controlled Linux environment using the
   CI Go version. Do not use a development-machine `make dev` artifact as the
   release artifact.
2. Record the source revision, Go version, build command, and SHA-256 value for
   `ela`, `ela-cli`, and `ela-dns`.
3. Sign the artifact manifest through the approved release process and verify
   the signature independently.
4. On each release candidate node, verify the binary version, mainnet network
   selection, startup logs, peer sync, and matching height/hash before it is
   considered upgraded.

## Replay and compatibility evidence

1. Start independent legacy and candidate nodes from clean data directories
   populated from the same trusted snapshot.
2. Replay through `E - 1`, compare block hashes at agreed checkpoints, and
   compare final height/hash.
3. Preserve commands, logs, source revisions, snapshot identifiers, and the
   first divergence if one occurs as immutable release evidence.
4. Do not treat replay after `E` as a compatibility test: the policy is
   intentionally restrictive at and after that height.

The shipped candidate disables `E` and `H` on testnet, regnet, and custom
networks after configuration parsing. Do not claim an activation rehearsal was
performed with the production binary on those networks. A rehearsal requires a
separately approved test-only harness or build.

## Bridge quarantine gate

1. Inventory queued deposits, withdrawals, refunds, and broadcaster queues.
2. Preserve queue evidence while preventing Arbiter workers from signing or
   broadcasting Type-7 or Type-81 transactions during `E` through `H - 1`.
3. Evidence that ESC inbound deposit observation and recharge processing are
   separately paused, buffered, or otherwise controlled.
4. Do not automatically resume bridge services at `H`. Require a separate,
   explicit authorization and a monitored controlled transaction flow.

## Rollout and attestation register

| Operator class | Exact artifact SHA/checksum | Started | Height/hash verified | Evidence | Status |
| --- | --- | --- | --- | --- | --- |
| Active/current-and-next BPoS |  |  |  |  |  |
| Standby BPoS |  |  |  |  |  |
| Primary template/AuxPoW |  |  |  |  |  |
| Failover template/AuxPoW |  |  |  |  |  |
| Monitoring/reference nodes |  |  |  |  |  |

Bitcoin ASIC miners do not run ELA transaction validation. The ELA-side
template and `submitauxblock` services they rely on must be upgraded and
attested.

## Countdown and activation monitoring

1. Before `E`, cross-check height and block hash across at least three upgraded
   independent reference nodes. `getblockcount` is a count; the latest height
   is `getblockcount - 1`.
2. At `E`, monitor accepted blocks, template creation/submission, BPoS progress,
   peer agreement, and reorg/fork alarms. Expected X-UTXO spend count is zero.
3. Do not broadcast exploit-shaped transactions on mainnet as a validation test.
4. Keep bridge exits and refunds paused throughout `E` through `H - 1`.
5. At `H`, continue detecting and rejecting generic X-UTXO spends. Resume only
   explicitly authorized and monitored Type-7/Type-81 operations after a
   separate GO decision.

## Abort and incident response

- Before `E`, stop rollout if a mandatory gate is incomplete. Do not attempt a
  local configuration override; issue a newly coordinated candidate with new
  future heights if the schedule must change.
- At or after `E`, do not unilaterally downgrade validators or template
  services. Preserve blocks, transactions, logs, source revisions, artifact
  checksums, and signatures, then invoke coordinated consensus incident
  response.
- If an X-UTXO spend appears at or after `E`, stop template creation/submission
  and bridge broadcasters immediately while preserving evidence.

## Publication and final sign-off

The internal notice must include the exact signed release identifier, artifact
checksums, `E`/`H`, upgrade deadline, bridge status, and escalation channel. A
public notice must describe scope and user impact without claiming that assets
are globally frozen, recovered, refunded, or that inbound deposits are stopped.

| Gate | Owner | Evidence reference | UTC | Approved |
| --- | --- | --- | --- | --- |
| Artifact integrity |  |  |  |  |
| Replay compatibility |  |  |  |  |
| Bridge quarantine |  |  |  |  |
| Validator/template rollout |  |  |  |  |
| Monitoring and incident response |  |  |  |  |
| Release GO decision |  |  |  |  |
