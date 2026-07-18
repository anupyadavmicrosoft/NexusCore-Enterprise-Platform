# API Specifications (NexusCore)

This document contains complete documentation of the REST and gRPC interfaces exposed by the NexusCore microservice mesh.

## 1. Authentication & Security Headers

All requests to internal endpoints (except public authentication paths) must include a cryptographically valid JSON Web Token (JWT) inside the `Authorization` header.

### 1.1 Required Header Configuration
```http
Authorization: Bearer <JWT_TOKEN_HERE>
X-Consumer-ID: usr_0192837465
X-Correlation-ID: tx_902831093123
Content-Type: application/json
```

---

## 2. API Endpoints Reference

### 2.1 Public Authentication Core

#### **POST** `/api/v1/auth/enroll`
Enrolls a new corporate credential entity inside the PostgreSQL system of record.

*   **Request Payload**:
    ```json
    {
      "username": "infra_controller_admin",
      "email": "admin@enterprise.nexus.internal",
      "password": "SecurePasswordLength99!",
      "organization": "Infrastructure-Operations"
    }
    ```
*   **Response Payload (`201 Created`)**:
    ```json
    {
      "status": "SUCCESS",
      "user_id": "usr_99812a83f211",
      "username": "infra_controller_admin",
      "enrolled_at": "2026-07-18T14:10:00Z"
    }
    ```

#### **POST** `/api/v1/auth/login`
Validates entity credentials and issues a multi-layered cryptographic authorization token.

*   **Request Payload**:
    ```json
    {
      "email": "admin@enterprise.nexus.internal",
      "password": "SecurePasswordLength99!"
    }
    ```
*   **Response Payload (`200 OK`)**:
    ```json
    {
      "access_token": "header.eyJzdWIiOiJhZG1pbi1wcmluY2lwYWwtOTl4IiwiZXhwIjoxNzg5MTIzNDU2fQ.signature",
      "token_type": "Bearer",
      "expires_in": 3600,
      "refresh_token": "rf_01a91e9202a39281"
    }
    ```

---

### 2.2 Ledger Transactions Core

#### **POST** `/api/v2/transactions`
Orchestrates a ledger credit or debit event. Triggers CQRS state mutation and Kafka event dispatch.

*   **Request Payload**:
    ```json
    {
      "account_id": "acc-9921-prod-core",
      "amount": 25000.50,
      "currency": "USD",
      "operation": "CREDIT"
    }
    ```
*   **Response Payload (`201 Created`)**:
    ```json
    {
      "tx_id": "tx-auto-908123",
      "account_id": "acc-9921-prod-core",
      "amount": 25000.50,
      "currency": "USD",
      "status": "COMMITTED",
      "correlation_id": "tx_902831093123",
      "committed_at": "2026-07-18T14:10:02Z"
    }
    ```

#### **GET** `/api/v2/accounts/{account_id}/history`
Fetches sanitized transaction records matching query parameter criteria.

*   **Parameters**:
    *   `query` (string, optional) - SQL-escaped filter string.
    *   `limit` (int, optional, default: `20`) - Page pagination bounds.
*   **Response Payload (`200 OK`)**:
    ```json
    [
      {
        "tx_id": "tx-auto-908123",
        "account_id": "acc-9921-prod-core",
        "amount": 25000.50,
        "currency": "USD",
        "status": "COMMITTED",
        "committed_at": "2026-07-18T14:10:02Z"
      }
    ]
    ```

---

## 3. Rate Limiting Specifications

Enforced at the Ingress Edge Layer (`api-gateway`) using a Sliding Window Log token bucket algorithm.
*   **Tier 1 Public Authenticated API**: `100` requests / minute per Client IP.
*   **Tier 2 Corporate API Access**: `2500` requests / minute per authenticated User ID token context.
*   **Burst Capacity**: Maximum burst of up to `2x` rate limits within standard `10-second` micro-windows before returning `429 Too Many Requests`.

---

## 4. Standardized Error Codes

NexusCore uses explicit JSON envelopes to convey exception details:

| HTTP Status | Application Error Code | Recovery Hint / Context |
| :--- | :--- | :--- |
| `400` | `INVALID_PAYLOAD_STRUCTURE` | Request JSON violates Swagger schema constraints. |
| `401` | `INVALID_CRYPTOGRAPHIC_SIGNATURE` | JWT validation signature failed or expired. |
| `403` | `INSUFFICIENT_SCOPE_PRIVILEGES` | User principal lacks required roles (e.g., `ClusterAdmin`). |
| `429` | `IP_RATE_LIMIT_EXCEEDED` | Client hit IP throughput limits. Implement back-off. |
| `422` | `NEGATIVE_BALANCE_WRITE_REJECTED` | Transaction cannot proceed due to insufficient funds. |
| `500` | `UPSTREAM_SERVICE_UNREACHABLE` | Downstream gRPC node timed out or crashed. |
