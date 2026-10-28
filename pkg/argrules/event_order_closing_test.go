package argrules

import (
	"encoding/json"
	"testing"
)

// closingPriceSchema is testbook_v1, except that a closing order keeps the
// price it was sent with.
const closingPriceSchema = "closingprice_v1"

func init() {
	read := func(args map[string]json.RawMessage) (EventOrder, OrderCostWhy) {
		o, why := testSchemaRead(args)
		if why.Valued() && o.Closing {
			if raw, ok := args["price"]; ok {
				p, ok := ReadArgDecimal(raw)
				if !ok {
					return EventOrder{}, WhyBadNumber
				}
				o.Price = p
			}
		}
		return o, why
	}
	if err := RegisterEventSchema(EventSchema{Name: closingPriceSchema, Fields: testSchemaFields, Read: read}); err != nil {
		panic(err)
	}
}

// A closing order gets every input check an opening order gets: the price
// bounds, the count rules and the refused shapes. Only the cost is skipped.
func TestEventOrderMaxCostClosingInputChecks(t *testing.T) {
	for _, c := range []struct {
		js  string
		why OrderCostWhy
	}{
		{`{"market":"M","action":"sell","count":"1","price":"1","tif":"ioc","reduce_only":true}`, WhyEventBadPrice},
		{`{"market":"M","action":"sell","count":"1","price":"1.00","tif":"ioc","reduce_only":true}`, WhyEventBadPrice},
		{`{"market":"M","action":"sell","count":"1","price":"0","tif":"ioc","reduce_only":true}`, WhyEventBadPrice},
		{`{"market":"M","action":"sell","count":"1","price":"-0.5","tif":"ioc","reduce_only":true}`, WhyEventBadPrice},
		{`{"market":"M","action":"sell","count":"1","price":"7","tif":"ioc","reduce_only":true}`, WhyEventBadPrice},
		{`{"market":"M","action":"sell","count":"0","tif":"ioc","reduce_only":true}`, WhyEventBadCount},
		{`{"market":"M","action":"sell","count":"0.001","tif":"ioc","reduce_only":true}`, WhyEventBadCount},
		{`{"market":"M","action":"sell","count":"10000000000","tif":"ioc","reduce_only":true}`, WhyEventTooManyContracts},
		{`{"market":"","action":"sell","count":"1","tif":"ioc","reduce_only":true}`, WhyEventNoMarket},
		{`{"market":"M","action":"sell","count":"1","tif":"gtc","reduce_only":true}`, WhyEventResting},
		{`{"market":"M","action":"sell","amount":"5","tif":"ioc","reduce_only":true}`, WhyEventByAmount},
		{`{"market":"M","action":"sell","count":"1","tif":"ioc","reduce_only":true,"replaces":"x"}`, WhyEventAmendOrReplace},
		{`{"orders":[],"reduce_only":true}`, WhyEventBatch},
		{`{"market":"M","action":"sell","count":"1","tif":"fok","reduce_only":true}`, WhyEventClosingUnsure},
		{`{"market":"M","action":"sell","count":"1","tif":"ioc","reduce_only":true,"extra":1}`, WhyEventUnknownField},
	} {
		v, why := EventOrderMaxCost(closingPriceSchema, evArgs(t, c.js))
		if why != c.why || v != (EventOrderValue{}) {
			t.Errorf("%s: %+v %q, want %q", c.js, v, why, c.why)
		}
	}
	for _, js := range []string{
		`{"market":"M","action":"sell","count":"3.5","price":"0.99","tif":"ioc","reduce_only":true}`,
		`{"market":"M","action":"sell","count":"3.5","price":"0.01","tif":"ioc","reduce_only":true}`,
		`{"market":"M","action":"sell","count":"3.5","tif":"ioc","reduce_only":true}`,
	} {
		v, why := EventOrderMaxCost(closingPriceSchema, evArgs(t, js))
		if !why.Valued() || !v.Closing || v.Units != 0 || v.Contracts != 350 {
			t.Errorf("%s: %+v %q", js, v, why)
		}
	}
	// The argument rule cap refuses it too: the cost cap is the only check
	// a closing order skips.
	rules, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"place_event_order","shapes":[{"event_order":{"schema":"closingprice_v1","max_cost":"1","max_contracts":"10"}}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if d := Check(rules, "place_event_order", eventCall(`{"market":"M","action":"sell","count":"1","price":"2","tif":"ioc","reduce_only":true}`)); d == nil {
		t.Fatal("cap passed a closing order priced at 2")
	}
	if d := Check(rules, "place_event_order", eventCall(`{"market":"M","action":"sell","count":"10","price":"0.5","tif":"ioc","reduce_only":true}`)); d != nil {
		t.Fatalf("cap refused a closing order inside the bounds: %+v", d)
	}
}
