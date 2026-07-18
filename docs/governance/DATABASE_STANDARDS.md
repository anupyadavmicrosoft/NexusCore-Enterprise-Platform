# NexusCore Database Standards

## 1. Relational Database Principles (PostgreSQL)

PostgreSQL is the primary transactional datastore for the NexusCore enterprise platform. It serves as the single source of truth for user states, authorizations, and core ledger metrics.

To maintain transaction speed, protect data integrity, and guarantee continuous service availability, all schema changes and database query structures must satisfy the guidelines defined in this document.

---

## 2. Schema Management & Migration Governance

### 2.1 Schema Versioning & Lock Step
*   **Declarative Schema**: Database schemas must be defined in version-controlled files under `internal/repository` (for Go microservices) or equivalent locations.
*   **Incremental Migration Scripts**: Raw schema mutations (e.g., `ALTER TABLE`, `CREATE INDEX`) are strictly prohibited in manual executions. All updates must go through versioned, incremental migration files executed by automated tools (such as Golang Migrate, Drizzle Migrator, or Flyway) in CD pipelines.
*   **Downward Migrations**: Every schema change file must feature both a `.up.sql` and a `.down.sql` script to allow safe rollbacks.

### 2.2 Schema Evolution & Backward Compatibility
To support continuous deployments with zero downtime, schema changes must be completely backward-compatible with the active, preceding version of the microservice.
*   **Column Deletion / Renaming**: You can **never** delete or rename a column in a single release. This requires a multi-step migration cycle:
    1.  *Release N*: Add the new column; modify code to write to both the old and new columns.
    2.  *Release N+1*: Run a background migration task to copy historical values from the old column to the new column; update code to read/write exclusively from the new column.
    3.  *Release N+2*: Safely remove references to the old column from the codebase, and execute a migration to drop the old column from the database.
*   **Default Values**: New columns added to existing tables must either be nullable or feature an explicit `DEFAULT` value to prevent database write errors from older, active microservice containers.

---

## 3. Query Optimization & Indexes

### 3.1 Index Creation Parameters
*   **No Sequential Scans in Production**: Every query executed in a high-traffic execution path must be backed by an appropriate index.
*   **Foreign Keys**: Every foreign key column must be covered by a separate index to avoid slow join operations and table-level locking cascades on delete actions.
*   **Index Overhead Minimization**: Do not index every column. Indexes speed up `SELECT` operations but degrade `INSERT`, `UPDATE`, and `DELETE` performance.
*   **Concurrent Creation**: All index creation statements in migration scripts must utilize the `CONCURRENTLY` keyword to prevent PostgreSQL from blocking read/write traffic to the target table:
    ```sql
    CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
    ```

### 3.2 Transaction Boundary Guidelines
*   **Keep Transactions Microscopic**: Never hold a database transaction open while waiting for external network requests, file I/O operations, or long-running CPU calculations. Transactions must consist strictly of fast SQL statements and close immediately.
*   **Isolation Levels**:
    *   **Read Committed**: Standard default level for general read/write consistency.
    *   **Serializable**: Mandatory for Ledger, balance transfers, and highly sensitive state mutations. If serializable transactions fail due to serialization conflicts, the code must catch the error and execute a retry loop with exponential backoff.

---

## 4. Connection Pooling & Resource Quotas

To prevent database starvation under high cluster concurrency:
*   **Explicit Pool Allocation Limits**: Connection limits must be explicitly declared upon client initialization. Hardcoding or using default unbound pools is strictly prohibited.
    ```go
    // Good
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(10)
    db.SetConnMaxLifetime(10 * time.Minute)
    db.SetConnMaxIdleTime(5 * time.Minute)
    ```
*   **Statement Timeouts**: Every database connection must specify a statement timeout (e.g., `statement_timeout = 3000` ms) to prevent runaway or lock-blocked queries from consuming worker threads indefinitely.

---

## 5. Caching Integration & Redis Conventions

To bypass heavy database read loads, microservices integrate Redis as a high-performance, in-memory caching layer:
*   **Strict TTL (Time-To-Live)**: All cached objects must feature an explicit TTL. Storing un-expiring items in Redis is prohibited, as it leads to memory exhaustion and stale data bugs.
*   **Key Namespacing**: Redis keys must utilize colon-separated hierarchical namespacing to prevent collisions across subsystems.
    *   *Format*: `service:domain:entity:id`
    *   *Example*: `auth:users:profile:99482`
*   **Cache-Aside Pattern**: Always fetch the key from Redis first. On a cache miss, read the source data from PostgreSQL, write it back to Redis with an appropriate TTL, and return the payload.
*   **Invalidation Events**: When an entity is mutated in PostgreSQL, publish an invalidation event on Kafka or directly delete the corresponding key in Redis to ensure cache consistency.
