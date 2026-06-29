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
