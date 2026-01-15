package identity

type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
)

func RoleFromString(v string) Role {
    switch v {
    case string(RoleAdmin):
        return RoleAdmin
    default:
        return RoleUser
    }
}
