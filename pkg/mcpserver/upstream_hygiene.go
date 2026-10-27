package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"
)

// What the proxy hands to an upstream it starts or calls: the environment of
// a stdio upstream's process, and the caller's SatGate token.

// stdioBaseEnv is the fixed list of variables of the proxy's environment
// every stdio upstream receives (each only if the proxy has it). It is what a
// program needs to start and run: the search path, the home and user, the
// locale, the temp dir, the time zone, the terminal type and the system's
// TLS trust store, plus the variables Windows needs to run a program. Any
// other variable reaches an upstream only if its configuration names it in
// EnvPassthrough or sets it in Env.
var stdioBaseEnv = [...]string{
	"PATH",
	"HOME",
	"USER",
	"LOGNAME",
	"SHELL",
	"LANG",
	"LANGUAGE",
	"LC_ALL",
	"LC_CTYPE",
	"LC_MESSAGES",
	"TMPDIR",
	"TMP",
	"TEMP",
	"TZ",
	"TERM",
	"SSL_CERT_FILE",
	"SSL_CERT_DIR",
	// Windows
	"SystemRoot",
	"SystemDrive",
	"ComSpec",
	"PATHEXT",
	"windir",
	"USERPROFILE",
	"APPDATA",
	"LOCALAPPDATA",
	"ProgramData",
}

// stdioEnv is the environment of a stdio upstream's process: stdioBaseEnv,
// then cfg.EnvPassthrough, each with the value lookup gives (a name lookup
// does not have is left out), then cfg.Env, which wins. The list is never nil
// (a nil exec.Cmd.Env would mean the proxy's whole environment) and is sorted
// so the same configuration always gives the same list.
func stdioEnv(cfg UpstreamConfig, lookup func(string) (string, bool)) []string {
	vars := make(map[string]string, len(stdioBaseEnv)+len(cfg.EnvPassthrough)+len(cfg.Env))
	pass := func(name string) {
		if name == "" || strings.ContainsAny(name, "=\x00") {
			return
		}
		if v, ok := lookup(name); ok {
			vars[name] = v
		}
	}
	for _, name := range stdioBaseEnv {
		pass(name)
	}
	for _, name := range cfg.EnvPassthrough {
		pass(name)
	}
	for k, v := range cfg.Env {
		vars[k] = v
	}
	env := make([]string, 0, len(vars))
	for k, v := range vars {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return env
}

// callerTokenKey carries the caller's SatGate token from the proxy to the
// upstream manager, apart from params, for an upstream configured with
// ForwardCallerToken.
type callerTokenKey struct{}

func withCallerToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, callerTokenKey{}, token)
}

func callerTokenFrom(ctx context.Context) string {
	t, _ := ctx.Value(callerTokenKey{}).(string)
	return t
}

// callerToken is the token the request presents, read the way authenticate
// reads it: params._meta.token, else the connection's bearer credential.
func callerToken(ctx context.Context, req *Request) string {
	if t := extractMetaToken(req.Params); t != "" {
		return t
	}
	return transportTokenFrom(ctx)
}

// withoutMetaToken returns params with the token removed from _meta. Member
// names are matched the way encoding/json matches them (case-insensitively),
// since that is how authenticate reads the token, so no spelling of it is
// passed on, and every _meta member is cleaned if the name repeats. A _meta
// left empty is removed. Every other member keeps its order and its value
// byte for byte. Params with no _meta member, or that are not a JSON object,
// are returned unchanged.
func withoutMetaToken(params json.RawMessage) json.RawMessage {
	members, ok := objectMembers(params)
	if !ok {
		return params
	}
	hasMeta := false
	for _, mb := range members {
		if strings.EqualFold(mb.name, "_meta") {
			hasMeta = true
			break
		}
	}
	if !hasMeta {
		return params
	}
	kept := members[:0:0]
	for _, mb := range members {
		if !strings.EqualFold(mb.name, "_meta") {
			kept = append(kept, mb)
			continue
		}
		inner, isObject := objectMembers(mb.value)
		if !isObject {
			kept = append(kept, mb) // not an object: no token can be read from it
			continue
		}
		innerKept := inner[:0:0]
		for _, im := range inner {
			if !strings.EqualFold(im.name, "token") {
				innerKept = append(innerKept, im)
			}
		}
		if len(innerKept) == len(inner) {
			kept = append(kept, mb)
			continue
		}
		if len(innerKept) == 0 {
			continue // _meta held only the token
		}
		kept = append(kept, objectMember{name: mb.name, value: writeObject(innerKept)})
	}
	return writeObject(kept)
}

// objectMember is one member of a JSON object: its decoded name and its value
// exactly as written.
type objectMember struct {
	name  string
	value json.RawMessage
}

// objectMembers lists the members of the JSON object raw, in order, repeated
// names included. ok is false when raw is not exactly one JSON object.
func objectMembers(raw json.RawMessage) (members []objectMember, ok bool) {
	if len(raw) == 0 {
		return nil, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, false
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, false
		}
		name, isString := tok.(string)
		if !isString {
			return nil, false
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, false
		}
		members = append(members, objectMember{name: name, value: value})
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, false
	}
	return members, true
}

// writeObject writes members as a JSON object, values as they are.
func writeObject(members []objectMember) json.RawMessage {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, mb := range members {
		if i > 0 {
			b.WriteByte(',')
		}
		name, _ := json.Marshal(mb.name)
		b.Write(name)
		b.WriteByte(':')
		b.Write(mb.value)
	}
	b.WriteByte('}')
	return b.Bytes()
}

// outboundParams is what is sent to the upstream named name: params without
// any _meta.token, and with the caller's token put back only when that
// upstream is configured with ForwardCallerToken.
func (m *UpstreamManager) outboundParams(ctx context.Context, name string, params json.RawMessage) json.RawMessage {
	params = withoutMetaToken(params)
	tok := callerTokenFrom(ctx)
	if tok == "" {
		return params
	}
	m.mu.Lock()
	cfg, ok := m.config[name]
	m.mu.Unlock()
	if !ok || !cfg.ForwardCallerToken {
		return params
	}
	if len(params) == 0 || string(params) == "null" {
		return injectMetaToken(json.RawMessage(`{}`), tok)
	}
	return injectMetaToken(params, tok)
}
