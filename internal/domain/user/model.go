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
}
