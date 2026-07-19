module github.com/nexuscore/identity-platform/identity-service

go 1.22

require (
	github.com/nexuscore/identity-platform/shared-jwt-library v0.0.0
	github.com/nexuscore/identity-platform/shared-oauth-library v0.0.0
	github.com/nexuscore/identity-platform/shared-security-library v0.0.0
)

replace (
	github.com/nexuscore/identity-platform/shared-jwt-library => ../shared-jwt-library
	github.com/nexuscore/identity-platform/shared-oauth-library => ../shared-oauth-library
	github.com/nexuscore/identity-platform/shared-security-library => ../shared-security-library
)
