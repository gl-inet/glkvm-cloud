package device

type Status string

const (
	StatusOnline   Status = "online"
	StatusOffline  Status = "offline"
	StatusDisabled Status = "disabled"
)

type Device struct {
    ID            int64
    Ddns          string
    Mac           string
    Name          string
    Description   string
    IP            string
    Client        string
    Model         string // hardware model reported by GL devices, "" for other hosts
    DeviceGroupID *int64 // nil means ungrouped (admin-only visibility)
    Status        Status
    LastSeenAt    *int64
}

// ListItem is a device plus its joined device-group name, returned by the
// server-side paginated list query so callers don't need a second lookup.
type ListItem struct {
    Device
    GroupName string
}

// ListQuery describes a server-side filtered/sorted/paginated device listing.
// Filtering, ordering (incl. online-first) and pagination all happen in SQL so
// large fleets don't require loading every row into memory per request.
type ListQuery struct {
    // RestrictGroups limits results to devices whose device_group_id is in this
    // set. nil means no restriction (admin: all devices, including ungrouped).
    RestrictGroups []int64
    // Search matches ddns/mac/ip/description as a case-insensitive substring.
    // Callers pass it already lowercased and with ':' stripped (to match the
    // colon-less MAC stored in the DB).
    Search     string
    Unassigned bool   // only devices with no device group
    Status     string // "" = any; otherwise online|offline|disabled
    SortBy     string // id|ip|mac|ddns|description|connectedTime|deviceGroupName
    Order      string // "asc" (default) or "desc"
    Page       int    // 1-based
    PageSize   int    // 0 => no limit (return all matching rows)
}
