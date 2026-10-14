# Raft Coordination Implementation Review and Fixes

Status: Completed
Author: VibeCoding

This document outlines the security and consistency issues identified during the review of the Raft-inspired coordination layer in PostgreBase, along with the proposed solutions and progress on implementing them.

## Identified Issues

### 1. Commit-Before-Quorum Bug
*   **Description:** In `vector/cluster.go`, the Leader applies a strict SQLite operation locally before ensuring replication to a majority quorum of peers. If quorum replication fails, the local state remains mutated while an error is returned to the client, leading to divergent states across the cluster.
*   **Status:** Resolved
*   **Fix:** Modified `ProposeReplicated` in `vector/cluster.go` so that if the operation is `Strict`, `replicateSyncQuorum` is invoked *before* `applyOperation` is called on the Leader. If the quorum replication fails, the Leader rolls back the log assignment (`lastLogIndex` and `logs` slice) and returns an error without modifying its local database. In addition, background retries in `retryFailedReplications` now only replay up to `appliedLogIndex` (the committed watermark) to avoid transmitting uncommitted log entries.

### 2. Memory-only Coordinator State Persistence (Deadlock on Restart)
*   **Description:** The coordinator's key state variables (`appliedLogIndex`, `lastLogIndex`, `term`) are kept in memory only. If a node restarts, its log sequence resets to 0, but its physical SQLite database still contains all the old collections and records. This causes gap-detection errors and permanently halts sync.
*   **Status:** Resolved
*   **Fix:** Added `AppliedLogIndex` and `LastLogIndex` to `Status` struct in `vector/manager.go`. Modified `snapshotLocked()` of `Manager` to retrieve these state variables from the `Coordinator` via accessors (`AppliedLogIndex()`, `LastLogIndex()`, `RaftTerm()`). In `NewCoordinator()`, we restore `appliedLogIndex`, `lastLogIndex`, and `term` from the loaded `Manager` status, and initialize `c.peerNextIndex` for peers to `lastLogIndex + 1`. We also trigger `c.manager.Persist()` inside `ApplyReplicated` and `ProposeReplicated` after these log sequences update to guarantee their immediate persistence to the `_pb_vector_params_` table on disk, eliminating desynchronization on node restarts.

### 3. Lexicographical Reclaim Bug (Outdated Leader Reclaims)
*   **Description:** Leadership is decided lexicographically by the smallest reachable node address. If a node with the smallest address (e.g. Node A) is temporarily partitioned and misses writes, it immediately reclaims leadership when it recovers. It then proposes lower log indices which followers silently ignore, causing permanent cluster desynchronization.
*   **Status:** Resolved
*   **Fix:** Added `LastLogIndex` to `Heartbeat`, `HeartbeatReply`, and `PeerState` structs to propagate and store log progress during peer-to-peer exchanges. Enhanced `recomputeLeaderLocked()` to perform a log-completeness safety check: any candidate (including self) is only eligible for election if its log index is greater than or equal to the maximum `LastLogIndex` seen from all currently reachable peers. If an outdated node recovers, it remains a Follower until its background replay catches up to the current Leader's log index, at which point it can safely and cleanly reclaim lexicographical leadership. This is fully validated in `TestCoordinatorOutdatedLeaderCannotReclaim`.

---

## Progress and Verification
All issues have been successfully resolved, and all tests in `vector` and `replication` packages are compiling and passing at 100%. The lightweight coordination layer is now highly resilient and safe against partition-induced brain-split, out-of-sync overwrites, and state-sequence gaps.

## Phase 4 Progress: Settings & Parameter Replication (Task 1)
*   **Description:** Intercept system configuration changes (stored under `models.ParamAppSettings` in the `_params` table) and replicate them across the SQLite cluster so that all nodes stay in sync with administrative updates.
*   **Status:** Completed
*   **Implementation:**
    *   Defined `OperationParamUpsert` and `OperationParamDelete` in `replication/operations.go` along with `ParamUpsertPayload` and `ParamDeletePayload` structs.
    *   Added helper functions `NewParamUpsertOperation` and `NewParamDeleteOperation` to wrap param state changes as strict replicated SQLite operations.
    *   Updated the `Apply` function in `replication/operations.go` to support applying `models.Param` updates directly using `dao.Save` / `dao.DeleteParam`.
    *   Intercepted the Settings updates in `forms/settings_upsert.go:Submit()`: if `form.app.IsSQLiteCluster()` is true, the settings are serialized, optionally encrypted, wrapped as a replicated `models.Param` operation, and proposed directly to the cluster coordinator.
    *   Wired a listener in `core/base.go`'s `Apply` callback closure: after an `OperationParamUpsert` is applied, if the param key matches `"settings"`, the node triggers `app.RefreshSettings()` to dynamically reinitialize and reload the new configuration parameters in memory, eliminating runtime drift.
    *   Verified the complete end-to-end parameter replication and deletion flow in `TestApplyParamUpsertReplicatesParam` under the `replication` package.

## Phase 4 Progress: Migrations Replication & Bootstrap Coordination (Task 2)
*   **Description:** Align database migration (DDL/DML) execution in SQLite cluster mode so that migrations are applied exactly once on the Leader, logged into the replication sequence, and cleanly replayed to all Followers, preventing local schema drifts.
*   **Status:** Completed
*   **Implementation:**
    *   Added the `OnApply` callback field to the `Runner` struct in `tools/migrate/runner.go` to capture successful migration applies post-transaction commit without locking database resources during network proposes.
    *   Wired `OnApply` inside `core/base.go` after coordinator startup on the **designated Leader** (the node with the lexicographically smallest address in the static peer layout). The designated Leader proposes a `"migration.apply"` operation to the cluster coordinator.
    *   Modified `apis/serve.go`'s startup logic: stand-alone servers run migrations at boot, but SQLite cluster instances skip startup migrations and wait for the coordinator to start.
    *   Implemented `IsDesignatedLeader()` in `core/app.go` and `core/base.go` via a highly-efficient linear search across `nodeAddr` and `clusterPeers`, avoiding external sorting package imports.
    *   Supported the `"migration.apply"` replicated operation in `replication/operations.go`: when followers receive it, they locate the target migration from `migrations.AppMigrations`, execute its `Up` function locally within `dao.RunInTransaction`, and log it to the local `_migrations` table.
    *   Validated the entire distributed migration flow in `TestApplyMigrationReplicates` under the `replication` package, ensuring 100% schema alignment across cluster nodes.

## Coordination Hardening: Membership & Snapshot Safety
*   **Description:** Close follow-up gaps in dynamic membership propagation and snapshot installation so cluster nodes can catch up safely without accepting stale state.
*   **Status:** Completed
*   **Implementation:**
    *   Wired `core.BaseApp` into `CoordinatorConfig.App` during vector initialization, enabling the existing snapshot send/install path in real SQLite cluster mode.
    *   Updated heartbeat processing to learn all advertised `Members`, initialize peer state, and lower `peerNextIndex` from heartbeat progress so recovered peers are replayed from their actual log position.
    *   Changed dynamic `AddPeer` initialization to start catch-up from log index `1` instead of assuming the new peer is already at the leader's latest log.
    *   Persisted locally advanced log indices used during bootstrap fallback writes.
    *   Added stale snapshot rejection in `InstallSnapshot()` so delayed or outdated snapshots cannot roll a follower back behind its current applied index or term.
    *   Added unit coverage for heartbeat member learning, new-peer catch-up initialization, and stale snapshot rejection.

## Follow-up Hardening: Build, Concurrency and Schema Safety
*   **Description:** Issues found while reviewing the coordination-hardening commit above; fixed together with the module/vendor wiring.
*   **Status:** Completed
*   **Implementation:**
    *   **Build unblocked.** `go.mod` kept an absolute developer-machine `replace` (`=> /home/free/src/vibecoding`) for the renamed SDK, so no fresh clone or CI run could compile. The replace is removed; `github.com/startvibecoding/mothx v1.1.60` resolves from the proxy and the vendor tree is regenerated (`GOFLAGS=-mod=mod go mod vendor`), matching the `-mod=vendor` used by `.github/workflows/check.yml`.
    *   **Snapshot transfer race.** `retryFailedReplications` spawned an unsynchronized goroutine per tick once `CoordinatorConfig.App` was wired in. Added a per-peer `snapshotInFlight` guard, switched to `os.MkdirTemp` for the snapshot file (the previous `UnixNano`-based name in `$TMPDIR` could collide), and made the post-send `peerNextIndex` write monotonic so a slow transfer cannot regress a peer that raced ahead via replication acks.
    *   **`InstallSnapshot` watermark safety.** The DB swap now runs outside `c.mu` (it closes and reopens the data connection, which previously blocked heartbeats and `View()`), watermarks are only ever raised rather than assigned, malformed snapshots (`lastLogIndex < appliedLogIndex`) are rejected before any I/O, and term-based rejection follows raft semantics: an older term is ignored only when it carries no log beyond the local one.
    *   **Heartbeat progress ordering.** `ReceiveHeartbeat` no longer assigns `state.LastLogIndex` directly; `addPeerLocked` owns that field and keeps it monotonic, so an out-of-order (lower) heartbeat cannot erase progress learned from an earlier beat.
    *   **`Manager.snapshotLocked` re-entrancy.** It wrote coordinator watermarks through `m.status` while callers held only `m.mu.RLock`, which is both a data race and a deadlock risk (the coordinator callbacks re-enter the manager). It now copies into a local `Status` first. Caught by `go test ./vector/... -race`.
    *   **`AdvanceIndexLocally` durability.** The bootstrap fallback persist is logged instead of discarded, since losing it desynchronizes replay offsets between nodes.
    *   **MySQL schema safety.** `addSchemaFieldColumn` normalizes field options via `InitOptions()` (so `ColDefinition()` and the multi-value check always agree) and drops the half-added column if the backfill or `MODIFY` fails, because MySQL DDL does not roll back with the surrounding transaction. Index validation moved into `validateCollectionIndexes`, called before any DDL in `SyncRecordTableSchema`, so a rejected index expression can no longer leave the records table with its previous indexes already dropped.
    *   **`dbutils.ParseIndex` panic.** The index-name group of `indexRegex` accepts zero characters, so `CREATE INDEX ON t (col)` indexed `nameParts[0]` on an empty slice and panicked. It now falls through to an invalid `Index` that `IsValid()` rejects.
    *   **Agent migration drift.** `CREATE TABLE IF NOT EXISTS` never re-runs, so installations that applied the earlier revision kept `TEXT` columns. Added idempotent `alignAgentColumnTypes` (MySQL only) plus a real SQLite execution test for the migration instead of asserting only the type-helper strings.
    *   **Documentation.** `AGENTS.md` gained a "SQLite Cluster Coordination" section recording the coordinator's non-raft caveats and the monotonicity invariants, the mothx rename, the MySQL 8.0.13+ expression-index requirement, and the `ui/dist` prerequisite for `go build`.
    * **Auth index names were never quoted.** `SyncRecordTableSchema` built the three auth unique indexes as bare identifiers (`CREATE UNIQUE INDEX _<uuid>_email_idx ...`). Because collection ids are UUIDs containing hyphens, SQLite parsed `-` as subtraction and every auth-collection creation failed with `near "-": syntax error` — a pre-existing bug masked until now because no test ever created an auth collection through a real DAO. All nine statements (MySQL / SQLite / Postgres branches) now wrap the generated name in `[[...]]` so `dbx.processSQL` quotes it. Covered by `TestSyncRecordTableSchemaAuthIndexesQuoting`, which also asserts that two records with empty e-mails coexist while duplicate non-empty e-mails still collide.
