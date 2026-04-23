package logging

func ResolveWithHostDefaults(policy Policy) Policy {
	resolved := ResolvePolicy(policy)
	if resolved.Mode != ModeHost {
		return resolved
	}
	resolved.Format = "json"
	if len(resolved.Sinks) == 0 {
		resolved.Sinks = []SinkType{SinkStdout}
	}
	return resolved
}
