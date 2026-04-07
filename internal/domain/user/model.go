package user

import (
    "rttys/internal/domain/identity"
)

type Status string

const (
    StatusActive   Status = "active"
    StatusDisabled Status = "disabled"
)

type User struct {
    ID           int64
    Username     string
    Email        string
    Description  string
    PasswordHash string
    Role         identity.Role
    Status       Status
    IsSystem     bool
    AuthProvider string // "local", "oidc", "ldap"
    ExternalSub  string // OIDC sub claim / LDAP user DN
    LastLoginAt  *int64 // unix seconds, nil if never
    TotpSecret   string // base32 secret; "" when 2FA not enabled
    TotpEnabled  bool
    CreatedAt    int64 // unix seconds
}
