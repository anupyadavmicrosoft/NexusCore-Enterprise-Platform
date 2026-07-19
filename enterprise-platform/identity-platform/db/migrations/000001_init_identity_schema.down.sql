-- ==============================================================================
-- NexusCore Enterprise Platform - Identity Platform Database Schema (Sprint 3)
-- File: /enterprise-platform/identity-platform/db/migrations/000001_init_identity_schema.down.sql
-- Description: Rollback script to clean up all schema artifacts created by the up migration.
-- ==============================================================================

-- 1. Remove Row Level Security Policies
DROP POLICY IF EXISTS tenant_users_isolation ON users;
DROP POLICY IF EXISTS tenant_organizations_isolation ON organizations;
DROP POLICY IF EXISTS tenant_audit_logs_isolation ON audit_logs;

-- 2. Drop Views
DROP VIEW IF EXISTS v_active_sessions;
DROP VIEW IF EXISTS v_user_permissions;

-- 3. Drop Stored Procedures
DROP PROCEDURE IF EXISTS record_failed_login_attempt(UUID, INT, INT);
DROP PROCEDURE IF EXISTS register_user_v1(UUID, UUID, VARCHAR, VARCHAR, OUT UUID);

-- 4. Drop Triggers & Related Functions
DROP TRIGGER IF EXISTS trigger_audit_logs_chain ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_tamper_chain_hash();

DROP TRIGGER IF EXISTS trigger_log_password_history ON users;
DROP FUNCTION IF EXISTS log_password_change();

DROP TRIGGER IF EXISTS trigger_update_sessions_timestamp ON sessions;
DROP TRIGGER IF EXISTS trigger_update_roles_timestamp ON roles;
DROP TRIGGER IF EXISTS trigger_update_users_timestamp ON users;
DROP TRIGGER IF EXISTS trigger_update_organizations_timestamp ON organizations;
DROP TRIGGER IF EXISTS trigger_update_tenants_timestamp ON tenants;

DROP FUNCTION IF EXISTS update_modified_column();

-- 5. Drop Tables (in correct dependency order)
DROP TABLE IF EXISTS password_history CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS login_history CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS devices CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- 6. Clean Up Extensions (Optional, only drop if no other apps rely on them)
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "ltree";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
