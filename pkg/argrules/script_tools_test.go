package argrules

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunsScriptsListAndForms(t *testing.T) {
	yes := []string{"ExecuteWixAPI", "executewixapi", "EXECUTEWIXAPI", "wix:ExecuteWixAPI", "wix/ExecuteWixAPI", "wix.ExecuteWixAPI", "wix_ExecuteWixAPI", "wix-executewixapi", "mcp_wix_ExecuteWixAPI",
		"SearchWixAPISpec", "searchwixapispec", "SEARCHWIXAPISPEC", "wix:SearchWixAPISpec", "wix/SearchWixAPISpec", "wix.SearchWixAPISpec", "wix_SearchWixAPISpec", "wix-searchwixapispec", "mcp_wix_SearchWixAPISpec"}
	for _, n := range yes {
		if !RunsScripts(n) {
			t.Errorf("RunsScripts(%q) = false, want true", n)
		}
	}
	no := []string{"", "CallWixSiteAPI", "ManageWixSite", "ListWixSites", "make_api_request", "place_equity_order", "MyExecuteWixAPI", "ExecuteWixAPIs", "ExecuteWix", "Execute", "SearchWixAPISpecs", "MySearchWixAPISpec", "SearchWix", "SearchWixAPI"}
	for _, n := range no {
		if RunsScripts(n) {
			t.Errorf("RunsScripts(%q) = true, want false", n)
		}
	}
	if len(ScriptTools()) == 0 {
		t.Fatal("empty list")
	}
	for _, st := range ScriptTools() {
		if st.Name == "" || len(st.Reason) < 20 {
			t.Errorf("entry %+v needs a name and a reason", st)
		}
		if !RunsScripts(st.Name) {
			t.Errorf("%s is on the list but RunsScripts is false", st.Name)
		}
	}
	// The returned list is a copy.
	l := ScriptTools()
	l[0].Name = "changed"
	if !RunsScripts("ExecuteWixAPI") {
		t.Fatal("editing the returned list changed the list")
	}
}

// AR3-2: Wix's documentation-search tool runs JavaScript too, so it is on the
// list by name, with its own reason.
func TestSearchWixAPISpecIsOnTheScriptToolList(t *testing.T) {
	var found bool
	for _, st := range ScriptTools() {
		if st.Name == "SearchWixAPISpec" {
			found = true
			if !strings.Contains(st.Reason, "JavaScript") || !strings.Contains(st.Reason, "sandbox") {
				t.Errorf("reason = %q", st.Reason)
			}
		}
	}
	if !found {
		t.Fatal("SearchWixAPISpec is not on the script-tool list")
	}
	rules, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"SearchWixAPISpec","shapes":[{"method":{"one_of":["GET"]}}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	e := ScriptToolRuleError(rules)
	if e == nil || e.Error() != "SearchWixAPISpec runs scripts, so a limit on it can't be enforced; leave it out of a limited key" {
		t.Fatalf("error = %v", e)
	}
	params, _ := json.Marshal(map[string]any{"name": "SearchWixAPISpec", "arguments": map[string]any{"code": "x", "reason": "y"}})
	if d := Check(rules, "SearchWixAPISpec", params); d == nil || d.Reason != ReasonScriptToolRule {
		t.Errorf("call-time denial = %+v", d)
	}
}

func TestScriptToolRuleErrorSentence(t *testing.T) {
	rules, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"]}}]},{"tool":"ExecuteWixAPI","shapes":[{"method":{"one_of":["GET"]}}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	e := ScriptToolRuleError(rules)
	if e == nil || e.Error() != "ExecuteWixAPI runs scripts, so a limit on it can't be enforced; leave it out of a limited key" {
		t.Fatalf("error = %v", e)
	}
	if ScriptToolRuleError(rules[:1]) != nil {
		t.Fatal("a rule on CallWixSiteAPI alone was refused")
	}
}

// A token built without the mint (the rule is in the caveats) is refused on
// every call, for the script tool and for any other tool.
func TestCheckRefusesEveryCallOnATokenWithAScriptToolRule(t *testing.T) {
	rules, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"ExecuteWixAPI","shapes":[{"method":{"one_of":["GET"]}}]},{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"]}}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"ExecuteWixAPI", "CallWixSiteAPI", "ListWixSites"} {
		params, _ := json.Marshal(map[string]any{"name": tool, "arguments": map[string]any{"method": "GET"}})
		d := Check(rules, tool, params)
		if d == nil || d.Reason != ReasonScriptToolRule {
			t.Errorf("%s: denial = %+v", tool, d)
		}
		if d != nil && strings.Contains(d.Reason, "GET") {
			t.Errorf("reason echoes an argument: %q", d.Reason)
		}
	}
}
