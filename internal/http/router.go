package http

import (
    "rttys/internal/domain/device"
    "rttys/internal/domain/permission"
    "rttys/internal/domain/user"
    "rttys/internal/http/handler"
    "rttys/internal/http/middleware"
    "rttys/internal/store/memory"
    "rttys/internal/store/sqlite"

    "context"

    "github.com/gin-gonic/gin"
)

type Deps struct {
    UserSvc      *user.Service
    PermSvc      *permission.Service
    DevSvc       *device.Service
    GroupRepo    *sqlite.GroupRepo
    SessionStore *memory.SessionStore
}

func NewRouter(d Deps) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(middleware.Trace())

    authH := handler.NewAuthHandler(d.UserSvc, d.SessionStore)
    meH := handler.NewMeHandler()
    devH := handler.NewDeviceHandler(d.DevSvc)
    dgH := handler.NewDeviceGroupHandler(d.GroupRepo)
    ugH := handler.NewUserGroupHandler(d.GroupRepo)

    userAdminH := handler.NewUserAdminHandler(d.UserSvc)
    ugAdminH := handler.NewUserGroupAdminHandler(
        func(ctx context.Context, name, desc string) (int64, error) { return d.GroupRepo.CreateUserGroup(ctx, name, desc) },
        func(ctx context.Context, id int64, name, desc string) error { return d.GroupRepo.UpdateUserGroup(ctx, id, name, desc) },
        func(ctx context.Context, id int64) error { return d.GroupRepo.DeleteUserGroup(ctx, id) },
    )
    dgAdminH := handler.NewDeviceGroupAdminHandler(
        func(ctx context.Context, name, desc string) (int64, error) { return d.GroupRepo.CreateDeviceGroup(ctx, name, desc) },
        func(ctx context.Context, id int64, name, desc string) error { return d.GroupRepo.UpdateDeviceGroup(ctx, id, name, desc) },
        func(ctx context.Context, id int64) error { return d.GroupRepo.DeleteDeviceGroup(ctx, id) },
    )

    // public
    r.POST("/api/login", authH.Login)

    // authed group
    api := r.Group("/api")
    api.Use(middleware.Auth(d.SessionStore, d.UserSvc, d.PermSvc))

    // auth
    api.POST("/logout", middleware.Require(permission.AuthWrite), authH.Logout)

    // me
    api.GET("/me", middleware.Require(permission.MeRead), meH.GetMe)

    // device scope list
    api.GET("/devices", middleware.Require(permission.DeviceRead), devH.ListDevices)

    // groups list
    api.GET("/device-groups", middleware.Require(permission.DeviceGroupRead), dgH.ListDeviceGroups)
    api.GET("/user-groups", middleware.Require(permission.UserGroupRead), ugH.ListUserGroups)

    // --- Admin CRUD ---
    api.GET("/users", middleware.Require(permission.UserRead), userAdminH.ListUsers)
    api.POST("/users", middleware.Require(permission.UserWrite), userAdminH.CreateUser)
    api.PUT("/users/:id", middleware.Require(permission.UserWrite), userAdminH.UpdateUser)
    api.DELETE("/users/:id", middleware.Require(permission.UserWrite), userAdminH.DeleteUser)

    api.POST("/user-groups", middleware.Require(permission.UserGroupWrite), ugAdminH.Create)
    api.PUT("/user-groups/:id", middleware.Require(permission.UserGroupWrite), ugAdminH.Update)
    api.DELETE("/user-groups/:id", middleware.Require(permission.UserGroupWrite), ugAdminH.Delete)

    api.POST("/device-groups", middleware.Require(permission.DeviceGroupWrite), dgAdminH.Create)
    api.PUT("/device-groups/:id", middleware.Require(permission.DeviceGroupWrite), dgAdminH.Update)
    api.DELETE("/device-groups/:id", middleware.Require(permission.DeviceGroupWrite), dgAdminH.Delete)

    return r
}
