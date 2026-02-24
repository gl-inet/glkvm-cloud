package handler

import (
    "context"
)

type UserGroupAdminHandler struct {
    create func(ctx context.Context, name, description string) (int64, error)
    update func(ctx context.Context, id int64, name, description string) error
    delete func(ctx context.Context, id int64) error
}

func NewUserGroupAdminHandler(
    create func(ctx context.Context, name, description string) (int64, error),
    update func(ctx context.Context, id int64, name, description string) error,
    del func(ctx context.Context, id int64) error,
) *UserGroupAdminHandler {
    return &UserGroupAdminHandler{create: create, update: update, delete: del}
}
