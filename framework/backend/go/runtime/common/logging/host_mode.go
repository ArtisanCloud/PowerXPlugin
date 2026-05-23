package logging

func ResolveWithHostDefaults(policy Policy) Policy {
	return ResolveWithHostMode(policy, IsHostProxyMode())
}

func ResolveWithHostMode(policy Policy, hostMode bool) Policy {
	resolved := ResolvePolicy(policy)
	if !hostMode {
		return resolved
	}
	// Host proxy mode is enforced by PowerX runtime:
	// mode=host, sinks=[stdout], format=json
	resolved.Mode = ModeHost
	resolved.Format = "json"
	resolved.Sinks = []SinkType{SinkStdout}
	resolved.AuthorizedExtraSinks = nil
	return resolved
}
