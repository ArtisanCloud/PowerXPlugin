package middleware

import "path"
import "strings"

type Permission struct{ Resource, Action string }
type RBACConfig struct {
	Enabled          bool
	DefaultDeny      bool
	SuperAdminRoles  []string
	RoutePermissions map[string]Permission // "METHOD:/api/v1/templates/*" -> {template,read}
	DelegateToPowerX bool
	PowerXIssuer     string
	PowerXAudience   string
}

func IsSuperAdmin(userRoles, superRoles []string) bool {
	if len(userRoles) == 0 || len(superRoles) == 0 {
		return false
	}
	s := map[string]struct{}{}
	for _, r := range userRoles {
		s[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}
	for _, r := range superRoles {
		if _, ok := s[strings.ToLower(strings.TrimSpace(r))]; ok {
			return true
		}
	}
	return false
}
func HasPerm(userPerms []string, need Permission) bool {
	if len(userPerms) == 0 {
		return false
	}
	want := strings.ToLower(strings.TrimSpace(need.Resource)) + ":" + strings.ToLower(strings.TrimSpace(need.Action))
	for _, p := range userPerms {
		p = strings.ToLower(strings.TrimSpace(p))
		switch {
		case p == want, p == "*", p == need.Resource+":*", strings.HasSuffix(p, ":*") && strings.TrimSuffix(p, ":*") == need.Resource:
			return true
		}
	}
	return false
}
func MatchRoute(method, reqPath string, table map[string]Permission) (Permission, bool) {
	if table == nil {
		return Permission{}, false
	}
	if perm, ok := table[method+":"+reqPath]; ok {
		return perm, true
	}
	for k, perm := range table {
		if i := strings.IndexByte(k, ':'); i >= 0 {
			m, pat := k[:i], k[i+1:]
			if (m == method || m == "*") && match(pat, reqPath) {
				return perm, true
			}
		}
	}
	return Permission{}, false
}
func match(pat, p string) bool { ok, _ := path.Match(pat, p); return ok }
