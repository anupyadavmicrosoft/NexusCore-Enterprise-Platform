-- ==============================================================================
-- NexusCore Enterprise Platform - Identity Platform Database Schema (Sprint 3)
-- File: /enterprise-platform/identity-platform/db/migrations/000001_init_identity_schema.up.sql
-- Description: Production-ready PostgreSQL schema including tables, constraints,
--              indexes, triggers, views, and stored procedures for multi-tenant IAM.
-- ==============================================================================

-- ------------------------------------------------------------------------------
-- 0. Prerequisites & Extensions
-- ------------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "ltree";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ------------------------------------------------------------------------------
-- 1. Base Shared Trigger Functions
-- ------------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ------------------------------------------------------------------------------
-- 2. Core Tables Definition
-- ------------------------------------------------------------------------------

-- 2.1 Tenants Table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE' CONSTRAINT chk_tenant_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'OFFBOARDING', 'TERMINATED')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trigger_update_tenants_timestamp
BEFORE UPDATE ON tenants
FOR EACH ROW EXECUTE FUNCTION update_modified_column();


-- 2.2 Organizations Table (with hierarchical ltree support)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    path ltree NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_tenant_org_path UNIQUE (tenant_id, path)
);

CREATE TRIGGER trigger_update_organizations_timestamp
BEFORE UPDATE ON organizations
FOR EACH ROW EXECUTE FUNCTION update_modified_column();


-- 2.3 Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CONSTRAINT chk_user_status CHECK (status IN ('PROVISIONED', 'PENDING', 'ACTIVE', 'LOCKED', 'SUSPENDED', 'ARCHIVED')),
    mfa_secret VARCHAR(128),
    mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    password_changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_tenant_email UNIQUE (tenant_id, email)
);

CREATE TRIGGER trigger_update_users_timestamp
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_modified_column();


-- 2.4 Roles Table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL represents system-wide global roles
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_tenant_role_name UNIQUE (tenant_id, name)
);

CREATE TRIGGER trigger_update_roles_timestamp
BEFORE UPDATE ON roles
FOR EACH ROW EXECUTE FUNCTION update_modified_column();


-- 2.5 Permissions Table
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(150) UNIQUE NOT NULL, -- e.g., 'tenant:billing:write'
    name VARCHAR(150) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- 2.6 User Roles Junction Table
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);


-- 2.7 Role Permissions Junction Table
CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);


-- 2.8 Devices Table
CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fingerprint_hash VARCHAR(64) NOT NULL,
    name VARCHAR(150) NOT NULL,
    os VARCHAR(100) NOT NULL,
    browser VARCHAR(100) NOT NULL,
    is_trusted BOOLEAN NOT NULL DEFAULT FALSE,
    last_active_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_device UNIQUE (user_id, fingerprint_hash)
);


-- 2.9 Refresh Tokens Table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    family_id UUID NOT NULL, -- Token Rotation Family ID (Replay Detection)
    replaced_by_token_id UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- 2.10 Sessions Table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash VARCHAR(64) UNIQUE NOT NULL,
    device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    payload JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trigger_update_sessions_timestamp
BEFORE UPDATE ON sessions
FOR EACH ROW EXECUTE FUNCTION update_modified_column();


-- 2.11 Audit Logs Table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'SUCCESS',
    payload JSONB NOT NULL DEFAULT '{}',
    client_ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    hash_chain_value VARCHAR(64), -- Secure cryptographically chained hash
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- 2.12 Login History Table
CREATE TABLE login_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    country VARCHAR(100),
    city VARCHAR(100),
    status VARCHAR(50) NOT NULL, -- 'SUCCESS', 'FAILURE_BAD_CREDENTIALS', 'FAILURE_MFA', 'LOCKED'
    failure_reason TEXT,
    device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- 2.13 API Keys Table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    prefix VARCHAR(16) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- 2.14 Password History Table
CREATE TABLE password_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- ------------------------------------------------------------------------------
-- 3. Optimized Database Indexes
-- ------------------------------------------------------------------------------

-- Indexes for performance tuning & high-throughput lookup queries

-- Hierarchical path queries GIST index
CREATE INDEX idx_organizations_path_gist ON organizations USING gist(path);

-- User matching indexes
CREATE INDEX idx_users_email_tenant ON users(tenant_id, email);
CREATE INDEX idx_users_org_id ON users(org_id);

-- Session token verification speedup
CREATE INDEX idx_sessions_token_hash ON sessions(session_token_hash);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- Refresh Token rotation family speedup
CREATE INDEX idx_refresh_tokens_family_hash ON refresh_tokens(family_id, token_hash);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Device pairing index
CREATE INDEX idx_devices_user_fingerprint ON devices(user_id, fingerprint_hash);

-- Audit log indexing
CREATE INDEX idx_audit_logs_tenant_created ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_user_event ON audit_logs(user_id, event_type);

-- Login history analytic speedups
CREATE INDEX idx_login_history_user_created ON login_history(user_id, created_at DESC);

-- API Keys fast match index
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);


-- ------------------------------------------------------------------------------
-- 4. Triggers & Constraints Enforcement Routines
-- ------------------------------------------------------------------------------

-- 4.1 Trigger: Automatically keep password history updated
CREATE OR REPLACE FUNCTION log_password_change()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') OR (OLD.password_hash <> NEW.password_hash) THEN
        INSERT INTO password_history (user_id, password_hash)
        VALUES (NEW.id, NEW.password_hash);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_log_password_history
AFTER INSERT OR UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION log_password_change();


-- 4.2 Trigger: Secure tamper-detection hashing chain for audit logs
CREATE OR REPLACE FUNCTION audit_logs_tamper_chain_hash()
RETURNS TRIGGER AS $$
DECLARE
    prev_hash VARCHAR(64);
BEGIN
    -- Fetch previous chain hash value
    SELECT hash_chain_value INTO prev_hash
    FROM audit_logs
    ORDER BY created_at DESC, id DESC
    LIMIT 1;

    IF prev_hash IS NULL THEN
        prev_hash := '0000000000000000000000000000000000000000000000000000000000000000';
    END IF;

    -- Concat and generate next SHA256 link in block
    NEW.hash_chain_value := encode(digest(prev_hash || NEW.event_type || NEW.status || COALESCE(NEW.user_id::text, '') || NEW.created_at::text, 'sha256'), 'hex');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_audit_logs_chain
BEFORE INSERT ON audit_logs
FOR EACH ROW EXECUTE FUNCTION audit_logs_tamper_chain_hash();


-- ------------------------------------------------------------------------------
-- 5. Stored Procedures & Transactional APIs
-- ------------------------------------------------------------------------------

-- 5.1 Stored Procedure: Register a new system user with standard structures
CREATE OR REPLACE PROCEDURE register_user_v1(
    p_tenant_id UUID,
    p_org_id UUID,
    p_email VARCHAR(255),
    p_password_hash VARCHAR(255),
    OUT r_user_id UUID
)
LANGUAGE plpgsql AS $$
BEGIN
    -- Check if tenant exists and is active
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = p_tenant_id AND status = 'ACTIVE') THEN
        RAISE EXCEPTION 'Target tenant does not exist or is suspended';
    END IF;

    -- Check if username is already bound in this tenant scope
    IF EXISTS (SELECT 1 FROM users WHERE tenant_id = p_tenant_id AND email = p_email) THEN
        RAISE EXCEPTION 'A user profile with this email already exists within the tenant scope';
    END IF;

    INSERT INTO users (tenant_id, org_id, email, password_hash, status)
    VALUES (p_tenant_id, p_org_id, p_email, p_password_hash, 'PENDING')
    RETURNING id INTO r_user_id;

    -- Log transaction event
    INSERT INTO audit_logs (tenant_id, user_id, event_type, action, status, payload, client_ip)
    VALUES (p_tenant_id, r_user_id, 'USER_PROVISION_SUCCESS', 'CREATE_USER', 'SUCCESS', jsonb_build_object('email', p_email), '127.0.0.1');
END;
$$;


-- 5.2 Stored Procedure: Increment failed logins and lock accounts dynamically
CREATE OR REPLACE PROCEDURE record_failed_login_attempt(
    p_user_id UUID,
    p_max_attempts INT,
    p_lock_duration_minutes INT
)
LANGUAGE plpgsql AS $$
DECLARE
    v_attempts INT;
BEGIN
    UPDATE users
    SET failed_login_attempts = failed_login_attempts + 1
    WHERE id = p_user_id
    RETURNING failed_login_attempts INTO v_attempts;

    IF v_attempts >= p_max_attempts THEN
        UPDATE users
        SET status = 'LOCKED',
            locked_until = CURRENT_TIMESTAMP + (p_lock_duration_minutes || ' minutes')::INTERVAL,
            failed_login_attempts = 0
        WHERE id = p_user_id;
    END IF;
END;
$$;


-- ------------------------------------------------------------------------------
-- 6. High-Performance Relational Views
-- ------------------------------------------------------------------------------

-- 6.1 View: User Permissions Aggregator Map
CREATE OR REPLACE VIEW v_user_permissions AS
SELECT 
    u.id AS user_id,
    u.tenant_id,
    u.email,
    r.id AS role_id,
    r.name AS role_name,
    p.code AS permission_code
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
JOIN role_permissions rp ON r.id = rp.role_id
JOIN permissions p ON rp.permission_id = p.id
WHERE u.status = 'ACTIVE';


-- 6.2 View: Active Session Monitoring Panel
CREATE OR REPLACE VIEW v_active_sessions AS
SELECT 
    s.id AS session_id,
    s.user_id,
    u.email,
    u.tenant_id,
    s.ip_address,
    s.user_agent,
    d.name AS device_name,
    d.os AS device_os,
    s.expires_at,
    (s.expires_at > CURRENT_TIMESTAMP) AS is_live
FROM sessions s
JOIN users u ON s.user_id = u.id
LEFT JOIN devices d ON s.device_id = d.id;


-- ------------------------------------------------------------------------------
-- 7. Row-Level Security (RLS) Policies
-- ------------------------------------------------------------------------------

-- Enforce strict data isolation at the engine layer
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_users_isolation ON users
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY tenant_organizations_isolation ON organizations
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY tenant_audit_logs_isolation ON audit_logs
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
