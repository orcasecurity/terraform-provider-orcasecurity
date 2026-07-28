package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// The group and user RBAC access endpoints share identical role+scope semantics;
// only the owner field name (group_id vs user_id), the nested owner key in list
// rows (group vs user), and the list transport differ. Everything below is the
// shared engine; group_access.go and user_access.go are thin type-bound wrappers.
//
// Neither endpoint exposes a /<id> route (by-id GET/PUT/DELETE 404) — the id
// travels in the request body, and reads are served by listing the collection.
//
// Group access is offset-pageable via OrcaLimitOffsetPaginator (limit +
// start_at_index query params, server max_limit 300, {total_items,data}
// envelope). User access is a single unpaginated GET that returns every row and
// ignores query params.
const (
	rbacAccessPageLimit = 300
	// rbacAccessMaxRows backstops the pagination loop: if the server ever stops
	// honoring start_at_index it would echo full pages forever, so we abort well
	// past any realistic assignment count rather than spin.
	rbacAccessMaxRows = 1_000_000
)

// rbacAccessEndpoint captures the per-endpoint differences the shared engine
// needs. ownerKey doubles as the collection label in error messages ("group"
// access / "user" access).
type rbacAccessEndpoint struct {
	path      string
	ownerKey  string // nested list-row object + payload field stem: "group" | "user"
	paginated bool
}

// rbacAccessRecord is the endpoint-agnostic canonical shape the engine operates
// on. GroupAccess/UserAccess convert to and from it at the package boundary.
type rbacAccessRecord struct {
	ID                string
	AllCloudAccounts  bool
	RoleID            string
	OwnerID           string // group id or user id
	CloudAccounts     []string
	ShiftleftProjects []string
	UserFilters       []string
}

// idRef unwraps the many {"id": "..."} objects the RBAC list rows nest.
type idRef struct {
	ID string `json:"id"`
}

type rbacAccessListRow struct {
	ID               string          `json:"id"`
	AllCloudAccounts bool            `json:"all_cloud_accounts"`
	Group            idRef           `json:"group"`
	User             idRef           `json:"user"`
	Role             idRef           `json:"role"`
	CloudAccounts    []idRef         `json:"cloud_accounts"`
	UserFilters      []string        `json:"user_filters"`
	ShiftleftRaw     json.RawMessage `json:"shiftleft_projects"`
}

func (row rbacAccessListRow) toRecord(ep rbacAccessEndpoint) rbacAccessRecord {
	ownerID := row.Group.ID
	if ep.ownerKey == "user" {
		ownerID = row.User.ID
	}
	cloudIDs := make([]string, 0, len(row.CloudAccounts))
	for _, ca := range row.CloudAccounts {
		if ca.ID != "" {
			cloudIDs = append(cloudIDs, ca.ID)
		}
	}
	return rbacAccessRecord{
		ID:                row.ID,
		OwnerID:           ownerID,
		RoleID:            row.Role.ID,
		AllCloudAccounts:  row.AllCloudAccounts,
		CloudAccounts:     cloudIDs,
		ShiftleftProjects: parseShiftleftProjectIDs(row.ShiftleftRaw),
		UserFilters:       row.UserFilters,
	}
}

// rbacAccessWire is the create/update request body. The owner id fields are
// omitempty so exactly one of group_id/user_id is sent per endpoint.
type rbacAccessWire struct {
	ID                string   `json:"id,omitempty"`
	AllCloudAccounts  bool     `json:"all_cloud_accounts"`
	RoleID            string   `json:"role_id"`
	GroupID           string   `json:"group_id,omitempty"`
	UserID            string   `json:"user_id,omitempty"`
	CloudAccounts     []string `json:"cloud_accounts"`
	ShiftleftProjects []string `json:"shiftleft_projects"`
	UserFilters       []string `json:"user_filters"`
}

func (r rbacAccessRecord) toWire(ep rbacAccessEndpoint) rbacAccessWire {
	w := rbacAccessWire{
		ID:                r.ID,
		AllCloudAccounts:  r.AllCloudAccounts,
		RoleID:            r.RoleID,
		CloudAccounts:     r.CloudAccounts,
		ShiftleftProjects: r.ShiftleftProjects,
		UserFilters:       r.UserFilters,
	}
	if ep.ownerKey == "user" {
		w.UserID = r.OwnerID
	} else {
		w.GroupID = r.OwnerID
	}
	return w
}

func parseShiftleftProjectIDs(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var strs []string
	if err := json.Unmarshal(raw, &strs); err == nil {
		return strs
	}
	var objs []idRef
	if err := json.Unmarshal(raw, &objs); err != nil {
		return nil
	}
	out := make([]string, 0, len(objs))
	for _, o := range objs {
		if o.ID != "" {
			out = append(out, o.ID)
		}
	}
	return out
}

func normalizeStringSliceForCompare(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func stringSliceSetEqual(a, b []string) bool {
	return slices.Equal(normalizeStringSliceForCompare(a), normalizeStringSliceForCompare(b))
}

// rbacAccessScopesMatch compares role and scope fields (not assignment id).
func rbacAccessScopesMatch(want, got rbacAccessRecord) bool {
	return want.RoleID == got.RoleID &&
		want.AllCloudAccounts == got.AllCloudAccounts &&
		stringSliceSetEqual(want.CloudAccounts, got.CloudAccounts) &&
		stringSliceSetEqual(want.ShiftleftProjects, got.ShiftleftProjects) &&
		stringSliceSetEqual(want.UserFilters, got.UserFilters)
}

func pickMatchingAccess(list []rbacAccessRecord, ownerID string, want rbacAccessRecord) *rbacAccessRecord {
	var matches []rbacAccessRecord
	for _, item := range list {
		if item.OwnerID != ownerID {
			continue
		}
		if !rbacAccessScopesMatch(want, item) {
			continue
		}
		matches = append(matches, item)
	}
	if len(matches) == 0 {
		return nil
	}
	if want.ID != "" {
		for _, m := range matches {
			if m.ID == want.ID {
				picked := m
				return &picked
			}
		}
	}
	picked := matches[0]
	return &picked
}

// parseAccessID extracts the assignment id from a create/update response,
// tolerating both the {"data": {...}} envelope and a bare object.
func parseAccessID(body []byte) string {
	var envelope struct {
		Data idRef `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Data.ID != "" {
		return envelope.Data.ID
	}
	var direct idRef
	if err := json.Unmarshal(body, &direct); err == nil {
		return direct.ID
	}
	return ""
}

// pageAllAccess returns every assignment in the collection, paging through the
// offset paginator when the endpoint supports it.
func (client *APIClient) pageAllAccess(ep rbacAccessEndpoint) ([]rbacAccessRecord, error) {
	var out []rbacAccessRecord
	fetched := 0
	for {
		path := ep.path
		if ep.paginated {
			q := url.Values{}
			q.Set("limit", strconv.Itoa(rbacAccessPageLimit))
			q.Set("start_at_index", strconv.Itoa(fetched))
			path += "?" + q.Encode()
		}
		resp, err := client.Get(path)
		if err != nil {
			return nil, err
		}
		var envelope struct {
			TotalItems int                 `json:"total_items"`
			Data       []rbacAccessListRow `json:"data"`
		}
		if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
			return nil, err
		}
		for _, row := range envelope.Data {
			out = append(out, row.toRecord(ep))
		}
		fetched += len(envelope.Data)
		if !ep.paginated || len(envelope.Data) == 0 || fetched >= envelope.TotalItems {
			return out, nil
		}
		if fetched >= rbacAccessMaxRows {
			return nil, fmt.Errorf(
				"list %s access: fetched %d rows without reaching total_items=%d; aborting to avoid an unbounded loop (server may be ignoring start_at_index)",
				ep.ownerKey, fetched, envelope.TotalItems)
		}
	}
}

// listAccessForOwner returns the assignments whose nested owner id equals ownerID.
func (client *APIClient) listAccessForOwner(ep rbacAccessEndpoint, ownerID string) ([]rbacAccessRecord, error) {
	all, err := client.pageAllAccess(ep)
	if err != nil {
		return nil, err
	}
	out := make([]rbacAccessRecord, 0, len(all))
	for _, r := range all {
		if r.OwnerID == ownerID {
			out = append(out, r)
		}
	}
	return out, nil
}

// findAccess resolves an assignment by scanning the collection: exact id match
// first, then role+scope as a fallback for when the id changed server-side.
func (client *APIClient) findAccess(ep rbacAccessEndpoint, assignmentID string, want rbacAccessRecord) (*rbacAccessRecord, error) {
	list, err := client.pageAllAccess(ep)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item.ID == assignmentID {
			picked := item
			return &picked, nil
		}
	}
	// Fallback: the id changed server-side (Orca merged or recreated the row).
	// Match by owner+role+scope instead. NOTE: with no by-id route to confirm a
	// deletion, this can adopt a sibling row of identical scope if the original
	// was truly deleted — that trade-off is deliberate; do not "fix" it into a
	// recreate, which is the re-creation bug WASP-1494 removed.
	want.ID = assignmentID
	return pickMatchingAccess(list, want.OwnerID, want), nil
}

func (client *APIClient) createAccess(ep rbacAccessEndpoint, rec rbacAccessRecord) (string, error) {
	resp, err := client.Post(ep.path, rec.toWire(ep))
	if err != nil {
		return "", err
	}
	id := parseAccessID(resp.Body())
	if id == "" {
		return "", fmt.Errorf("create %s access: could not parse assignment id from response: %s", ep.ownerKey, string(resp.Body()))
	}
	return id, nil
}

// updateAccess PUTs the assignment (id in the body) then re-reads the canonical
// row, since the PUT response nests owner/role rather than returning the flat shape.
func (client *APIClient) updateAccess(ep rbacAccessEndpoint, rec rbacAccessRecord) (*rbacAccessRecord, error) {
	if rec.ID == "" {
		return nil, fmt.Errorf("update %s access: id is required", ep.ownerKey)
	}
	if _, err := client.Put(ep.path, rec.toWire(ep)); err != nil {
		return nil, err
	}
	refreshed, err := client.findAccess(ep, rec.ID, rec)
	if err != nil {
		return nil, err
	}
	if refreshed != nil {
		return refreshed, nil
	}
	return &rec, nil
}

// deleteAccess removes an assignment (id carried in the body); a 404 is treated
// as already-gone.
func (client *APIClient) deleteAccess(ep rbacAccessEndpoint, id string) error {
	_, err := client.DeleteWithBody(ep.path, map[string]string{"id": id})
	if err != nil && strings.Contains(err.Error(), "status: 404") {
		return nil
	}
	return err
}
