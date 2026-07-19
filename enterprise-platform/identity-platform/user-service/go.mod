module github.com/nexuscore/identity-platform/user-service

go 1.22

require (
	github.com/nexuscore/identity-platform/shared-auth-library v0.0.0
	github.com/nexuscore/identity-platform/shared-security-library v0.0.0
)

replace (
	github.com/nexuscore/identity-platform/shared-auth-library => ../shared-auth-library
	github.com/nexuscore/identity-platform/shared-security-library => ../shared-security-library
)
