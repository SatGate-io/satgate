package argrules

import "strings"

// Tools that run scripts cannot carry a limit.
//
// A rule looks at the top-level arguments of one tools/call. A tool that takes
// a program as an argument (a script, a query in a general language, a chain of
// API calls) does whatever the program says, and no field next to the program
// describes that. A rule such as "method is GET" on such a tool would be
// accepted, shown as a limit, and enforced on a field the script ignores. So a
// rule on one of these tools is never minted, never shown as a limit, and a
// token that carries one is refused at tools/call (fail closed: the rule cannot
// be enforced).
//
// This is the one list. The gateway mint and delegation, the dashboard (through
// a JSON copy kept equal by a drift test) and the runtime all read it.

// ScriptTool names one tool that runs scripts, and why it is on the list.
type ScriptTool struct {
	Name   string
	Reason string
}

var scriptTools = []ScriptTool{
	{
		Name:   "ExecuteWixAPI",
		Reason: "takes a JavaScript function as its code argument and runs it against the Wix APIs; the script can chain any call, including writes and deletes, whatever the top-level method field says",
	},
	{
		Name:   "SearchWixAPISpec",
		Reason: "Wix's documentation-search tool; it takes JavaScript in its code argument (with a reason field) and runs it in a sandbox, so what it does is set by the script, not by any top-level field",
	},
}

// ScriptTools returns the list, a copy the caller may keep.
func ScriptTools() []ScriptTool {
	out := make([]ScriptTool, len(scriptTools))
	copy(out, scriptTools)
	return out
}

// ScriptToolNames returns the names on the list, in list order.
func ScriptToolNames() []string {
	out := make([]string, 0, len(scriptTools))
	for _, t := range scriptTools {
		out = append(out, t.Name)
	}
	return out
}

// RunsScripts reports whether toolName is a tool that runs scripts.
//
// The form matched is the name an upstream advertises and a token scope stores:
// Cloud MCP does not prefix tool names with the service, a selected-tools key
// stores "ExecuteWixAPI" as written, and a rule names the same string. The match
// is on the whole name, ignoring ASCII letter case, and also on a name that ends
// in a listed name after one separator (":", "/", ".", "_" or "-"), so a service
// qualified form such as "wix:ExecuteWixAPI" or "wix_executewixapi" is caught
// too. It is meant to over-match: a false positive only refuses a limit on a
// tool that is probably the same tool.
func RunsScripts(toolName string) bool {
	if toolName == "" {
		return false
	}
	for _, t := range scriptTools {
		if strings.EqualFold(toolName, t.Name) {
			return true
		}
		n := len(toolName) - len(t.Name)
		if n > 0 && strings.EqualFold(toolName[n:], t.Name) && strings.ContainsRune(":/._-", rune(toolName[n-1])) {
			return true
		}
	}
	return false
}

// ScriptToolRefusal is the sentence shown when a limit is asked for on a tool
// that runs scripts. It is the same sentence in the gateway and the dashboard.
func ScriptToolRefusal(toolName string) string {
	return toolName + " runs scripts, so a limit on it can't be enforced; leave it out of a limited key"
}

// ScriptToolRuleError returns the first rule in rules that names a tool that
// runs scripts, as an error carrying ScriptToolRefusal, or nil.
func ScriptToolRuleError(rules []Rule) error {
	for _, r := range rules {
		if RunsScripts(r.Tool) {
			return &ScriptToolError{Tool: r.Tool}
		}
	}
	return nil
}

// ScriptToolError says a rule names a tool that runs scripts.
type ScriptToolError struct{ Tool string }

func (e *ScriptToolError) Error() string { return ScriptToolRefusal(e.Tool) }
