# Event Contract Order Limits (developer reference)

This page is for developers who build a gateway on the MCP packages
(`pkg/argrules`, `pkg/toolcontrols`, `pkg/mcpserver`). It lists the registry,
the token keys and the refusal reasons for limits on event contract orders:
orders on two-outcome markets whose contracts settle at 1 or 0.

Nothing here names a service. A token never says which service an order tool
belongs to. It names a **value schema**, and the program that runs the
gateway registers each schema it supports. This repository registers none,
and `satgate-mcp` as built from it refuses to start with event contract
limits configured (see [Self-hosted](#self-hosted-satgate-mcp)).

## Valuation

Every event control uses one function, `argrules.EventOrderMaxCost(schema,
args)` (and `EventOrderMaxCostOfCall` for a whole `tools/call`). It returns
the most an order can cost:

- contracts × price per contract, when the order gives a price that bounds
  it;
- contracts × 1, when it does not;
- 0, when the schema says the order can only close a position (`Closing`).

The result is exact and rounded **up** once to whole 10^-8 units. An order
that can open exposure is never valued at 0. The contract count is kept in
hundredths of a contract.

A call is refused, never valued, when:

- the schema is not registered in the process;
- an argument is not on the schema's field list (a letter-case twin counts
  as not on the list);
- the market id is not 1 to 64 printable ASCII characters with no space;
- the contract count is not above zero with at most 2 decimals, or is
  10^10 or more;
- the price, when given, is not strictly between 0 and 1;
- the schema's reader refuses the shape. Readers use the fixed reasons below
  for resting orders, orders sized by amount, amend or replace, batches, and
  a close that cannot be guaranteed;
- the cost is 10^10 or more (opening orders only; a closing order has no
  cost).

Every check in this list except the last applies to closing orders exactly as
to opening orders, including the price bound when the reader returns a price.
A closing order skips only the cost: it is valued at 0, always passes
`max_cost`, and is not counted against day or week limits, approvals or
per-event totals.

## Registry

```go
// The host program registers each schema once, before it parses any token.
err := argrules.RegisterEventSchema(argrules.EventSchema{
    Name:   "example_v1",                 // [a-z][a-z0-9_]{0,39}
    Fields: []string{"market", "count", "price", "time_in_force"},
    Read: func(args map[string]json.RawMessage) (argrules.EventOrder, argrules.OrderCostWhy) {
        // Read only args. Return an EventOrder{Market, Contracts, Price, Closing}
        // or one of the fixed reasons (argrules.WhyEvent..., WhyUnreadable,
        // WhyBadNumber, WhyNotAboveZero).
    },
})

// And a market lookup for the same schema.
err = mcpserver.RegisterEventMarketResolver("example_v1", resolver)
```

- A name can be registered once. Registering it again is an error, so a
  schema's meaning cannot change under tokens that already name it. A changed
  reading gets a new name.
- A reader may return only the fixed reasons. Any other text is replaced by
  "it could not be read". A reader that panics refuses the call.
- `argrules.ReadArgString`, `ReadArgDecimal` and `ReadArgBool` read
  arguments the way argument rules do. No float64 is used.
- `EventMarketResolver.ResolveEventMarket(ctx, tenantID, schema, market)`
  returns an `EventMarket{Event, Series, Category, Binary, MaxValueOne,
  Combo, Tradable, ClosesAt}`, `ErrEventMarketNotFound`, or an error. It must
  read only data the agent cannot influence.

## Token keys

All keys are optional. A key names a schema that must be registered in the
process that reads the token; if it is not, the document does not parse and
the token is refused. Releases before these keys refuse every document that
carries one (unknown condition or key), so an older verifier denies the token
instead of dropping the limit.

### Per-order cap and contracts cap (argument rule)

```json
{"v":1,"rules":[{"tool":"place_event_order","shapes":[
  {"event_order":{"schema":"example_v1","max_cost":"25","max_contracts":"100"}}]}]}
```

- `event_order` is a reserved field name. It is read as this condition only
  when it carries `schema`. Without `schema`, a condition on an argument
  named `event_order` keeps its old meaning.
- At least one of `max_cost` (above 0, at most 10 digits before and 8 after
  the point) and `max_contracts` (above 0, at most 10 digits before and 2
  after the point).
- The call passes when value ≤ `max_cost` and contracts ≤ `max_contracts`.
  A closing order always passes `max_cost` and is still held to
  `max_contracts`.
- Refusal code: `TOOL_ARGUMENT_DENIED`, as for other argument rules.

### Day and week limits, approvals

The existing keys, with `"value":"event_max_cost"` and `"schema"`:

```json
{"tools":["place_event_order"],"field":"dollar_amount","max":"50","window":"day","tz":"America/New_York","value":"event_max_cost","schema":"example_v1"}
{"tools":["place_event_order"],"field":"dollar_amount","above":"20","value":"event_max_cost","schema":"example_v1"}
```

- `field` must be `dollar_amount`. It is not read.
- `schema` is required with `event_max_cost` and refused with any other
  `value`, or with no `value`.
- `symbols` is refused with `event_max_cost`.
- A closing order is not counted against a limit and is never held for
  approval.

### Allowed categories or series

```json
{"v":1,"event_markets":[{"tools":["place_event_order"],"schema":"example_v1",
  "categories":["ECONOMICS"],"series":["<series id>"]}]}
```

- A call passes an entry when the looked-up series is listed (exact match) or
  the looked-up category is listed. Categories are compared after trimming
  ASCII white space and ASCII upper-casing; they must be written that way.
- Each list is optional, at least one is given, and each holds 1 to 32
  entries of 1 to 64 printable ASCII characters.
- Every entry that covers the tool must pass. There is no block list.
- At most 8 entries per token.

### Total on any one event

```json
{"v":1,"event_stakes":[{"tools":["place_event_order"],"schema":"example_v1","max":"40","reset":"manual"}]}
```

- Sums the values of the orders counted on one event, across every market of
  that event. A closing order is not counted.
- `reset` must be `manual`. Totals are never reset by time. The host program
  resets them with `EventStakeStore.ResetEvents(ctx,
  mcpserver.EventStakeKey(tenantID, limit))`.
- Each reservation has its own identity, returned by `ReserveEvent` and bound
  to the generation of the totals; `ResetEvents` starts a new generation. A
  give-back releases only the reservation it names. A reservation made
  before a reset is dropped by it, and giving it back afterwards changes
  nothing, so it never lowers a total counted after the reset.
- A counter holds at most 10,000 events. A new event past that is refused
  until the totals are reset.
- At most 8 entries per token.

## The gate

`mcpserver.EventGate` runs first in `ChainGates(event, spend, approval)`. For
a call that an event control covers, it checks in this order:

1. The call is valued under the one schema the token's event controls name
   for the tool. Two different schemas refuse the call.
2. The tool's service (`ToolOrigin.Service`) is the one bound to the schema
   (`NewEventGate(store, map[schema]service)` or `BoundUpstream`).
3. The market is found, has an event, has two outcomes, a contract on it is
   worth at most 1, it is not a combination of markets, and it is open for
   trading (`Tradable`, and `ClosesAt` not passed). The lookup has a time
   limit (`EventGate.LookupTimeout`, default 5 seconds). An answer that comes
   after that deadline, or after the call's context is done, is not used,
   even when the lookup returned no error: the call is refused with
   `tool_controls_unavailable` before anything is reserved or sent.
4. Every `event_markets` entry allows it.
5. Every `event_stakes` entry has room. The value is reserved and given back
   only when the call is known not to have run (the spending limit rule).

The store must implement `toolcontrols.EventStakeStore`.
`toolcontrols.MemoryStore` does.

A gateway whose gate does not enforce event controls (no `EventGate` in the
chain, so `EventControlGate` is not implemented) refuses every call on a token
whose control words carry an event control, and every call to a tool that an
`event_order` rule on the token names. `TokenInfo.NeedsCallGate` is true for any event control,
including an argument rule that has only an `event_order` cap.

## Refusals

Codes:

| code | when |
|---|---|
| `TOOL_ARGUMENT_DENIED` | per-order cap or contracts cap |
| `TOOL_SPEND_LIMIT_DENIED` | the order cannot be valued; a day, week or per-event total would be passed; per-event totals are full; two schemas; the call's name or layout cannot be matched |
| `EVENT_MARKET_NOT_ALLOWED` | not on an allow list; market not found, no event, not two outcomes, worth more than 1, a combination, not open; schema not bound to the tool's service; no market lookup registered |
| `APPROVAL_REQUIRED`, `APPROVAL_CHECK_FAILED` | as for other approvals |
| `tool_controls_unavailable` | lookup error, or a lookup answer after its deadline, store error, store without per-event totals, tool table unavailable, no event gate installed |

Reasons an order cannot be valued (each follows "this token counts each order
at the most it can cost, and "):

- its value schema is not available here
- it carries an argument its value schema does not list
- it does not name its market as its value schema requires
- its contract count is not a number above zero with at most 2 decimals
- its contract count is too large to count
- its price is not a number between 0 and 1
- an order that can stay open after the day it was placed (good till cancelled, or expiring later) cannot be counted
- an order sized by an amount instead of a contract count cannot be counted
- an order that amends or replaces another order cannot be counted
- a batch of orders cannot be counted
- it asks to only close a position in a way that cannot be guaranteed
- it could not be read
- a number in it is not a plain number
- a number in it is not above zero
- its cost is too large to count

Gate messages (each starts with the tool name and ": "):

- this order's market is not in a category or series this token allows, so it was not sent
- this order's market could not be found, so the call was not sent
- this order's market has no event, so the call was not sent
- this order's market is not a two-outcome market, so the call was not sent
- a contract on this order's market can be worth more than 1, so the call was not sent
- this order's market combines other markets, so the call was not sent
- this order's market is not open for trading, so the call was not sent
- this order's value schema is not bound to the service this tool comes from, so the call was not sent
- markets for this order's value schema cannot be looked up here, so the call was not sent
- this token values this tool under more than one value schema, so the call was not sent
- this token's limit is $<max> in total on any one event, counting each order at the most it can cost; this order would go over it
- this token's per-event totals hold too many events to count another; they must be reset first, so the call was not sent

No refusal includes the market, event, series, category, contract count or
price.

Receipts: in this version, event controls add no receipt fields. A refusal
carries its code, and for a per-event total, the limit.

## Self-hosted (satgate-mcp)

```yaml
controls:
  store: memory            # counters in this process's memory; they start again on restart
  eventContracts:          # needs a program that registers these schemas
    - schema: example_v1
      upstream: my-upstream
```

- With no `controls` section nothing changes: a token that carries a control
  is refused on every call.
- With `store: memory`, `satgate-mcp` installs `ChainGates(event, spend,
  approval)` over one memory store. Day and week limits are enforced. A
  shared budget is refused. A call an approval rule would hold is refused,
  because there is no approval store.
- Each `eventContracts` entry must name a schema and a market lookup that the
  program registered. `satgate-mcp` as built from this repository registers
  none, so a config with any entry refuses to start with:
  `controls.eventContracts: value schema "<name>" is not registered by this
  program (registered: none); event contract limits need a host program that
  registers its value schemas, so the gateway was not started`.
- `mcpserver.ControlGateFromConfig(cfg)` builds the same gate for a host
  program that registers its schemas first.
