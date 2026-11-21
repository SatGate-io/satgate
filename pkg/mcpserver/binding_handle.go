package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

type ObservationState string

const (
	ObservationAvailable     ObservationState = "available"
	ObservationPending       ObservationState = "pending"
	ObservationUnknown       ObservationState = "unknown"
	ObservationNotDispatched ObservationState = "not_dispatched"
)

type HandleRef struct {
	ID   string `json:"id"`
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

type BindingHandle struct {
	Schema           string           `json:"schema"`
	Decision         HandleRef        `json:"decision"`
	Observation      *HandleRef       `json:"observation,omitempty"`
	ObservationState ObservationState `json:"observation_state"`
	owned            bool
}

const BindingHandleSchema = "satgate.binding_handle.v1"

func NewBindingHandle(decision HandleRef, state ObservationState, observation *HandleRef) (*BindingHandle, error) {
	if !validHandleRef(decision) {
		return nil, fs.ErrInvalid
	}
	switch state {
	case ObservationAvailable:
		if observation == nil || !validHandleRef(*observation) {
			return nil, fs.ErrInvalid
		}
		copy := *observation
		observation = &copy
	case ObservationPending, ObservationUnknown, ObservationNotDispatched:
		if observation != nil {
			return nil, fs.ErrInvalid
		}
	default:
		return nil, fs.ErrInvalid
	}
	return &BindingHandle{Schema: BindingHandleSchema, Decision: decision, Observation: observation, ObservationState: state, owned: true}, nil
}

func validHandleRef(ref HandleRef) bool {
	for _, s := range []string{ref.ID, ref.Hash} {
		if len(s) == 0 || len(s) > 256 || strings.ContainsFunc(s, func(r rune) bool {
			return unicode.IsSpace(r) || unicode.IsControl(r)
		}) {
			return false
		}
	}
	if len(ref.URL) > 2048 {
		return false
	}
	u, err := url.Parse(ref.URL)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil
}

func (h *BindingHandle) IsOwned() bool { return h != nil && h.owned }

func (h *BindingHandle) UnmarshalJSON(data []byte) error {
	type wireHandle BindingHandle
	var decoded wireHandle
	h.owned = false
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*h = BindingHandle(decoded)
	return nil
}

type BindingHandleProvider interface {
	HandleFor(ctx context.Context, req *Request) (*BindingHandle, error)
}

func (p *Proxy) SetBindingProvider(pvd BindingHandleProvider) {
	if pvd == nil {
		p.bindingProvider.Store(nil)
		return
	}
	p.bindingProvider.Store(&pvd)
}

func ApplyBindingHandle(result json.RawMessage, evidence *MCPEvidence, handle *BindingHandle) (json.RawMessage, bool) {
	if !handle.IsOwned() {
		return result, false
	}
	obj, ok := parseBindingObject(result)
	if !ok {
		return result, false
	}
	publicHandle, err := json.Marshal(handle)
	if err != nil {
		return result, false
	}
	evidenceData := map[string]any{"binding": json.RawMessage(publicHandle)}
	mergeEvidenceData(evidenceData, evidence)
	evidenceJSON, err := json.Marshal(evidenceData)
	if err != nil {
		return result, false
	}
	brokerResult := obj.field("structuredContent")
	if brokerResult == nil {
		brokerResult = obj.field("content")
	}
	if brokerResult == nil {
		brokerResult = json.RawMessage(`null`)
	}
	structured := append([]byte(`{"broker_result":`), brokerResult...)
	structured = append(structured, `,"satgate_receipt":`...)
	structured = append(structured, publicHandle...)
	structured = append(structured, '}')
	return obj.with(
		bindingJSONEdit{key: "structuredContent", value: func(json.RawMessage) []byte { return structured }},
		bindingJSONEdit{key: "_meta", value: func(raw json.RawMessage) []byte {
			meta, ok := parseBindingObject(raw)
			if !ok {
				meta = bindingJSONObject{raw: json.RawMessage(`{}`), close: 1}
			}
			return meta.with(bindingJSONEdit{key: "satgate_evidence", value: func(json.RawMessage) []byte { return evidenceJSON }})
		}},
	), true
}

const bindingRefSchema = `{"type":"object","additionalProperties":false,"required":["id","hash","url"],"properties":{"id":{"type":"string"},"hash":{"type":"string"},"url":{"type":"string"}}}`

const bindingReceiptSchema = `{"type":"object","additionalProperties":false,"required":["schema","decision","observation_state"],"properties":{"schema":{"const":"` + BindingHandleSchema + `"},"decision":` + bindingRefSchema + `,"observation":` + bindingRefSchema + `,"observation_state":{"enum":["` + string(ObservationAvailable) + `","` + string(ObservationPending) + `","` + string(ObservationUnknown) + `","` + string(ObservationNotDispatched) + `"]}}}`

func ToolWithBindingSchema(tool json.RawMessage) (json.RawMessage, bool) {
	obj, ok := parseBindingObject(tool)
	if !ok {
		return tool, false
	}
	for _, field := range obj.fields {
		if field.key != "outputSchema" {
			continue
		}
		schema, _ := parseBindingObject(obj.raw[field.start:field.end])
		for _, prop := range schema.fields {
			if prop.key != "properties" {
				continue
			}
			props, _ := parseBindingObject(schema.raw[prop.start:prop.end])
			if props.field("broker_result") != nil || props.field("satgate_receipt") != nil {
				return tool, false
			}
		}
	}
	return obj.with(bindingJSONEdit{key: "outputSchema", value: func(broker json.RawMessage) []byte {
		if broker == nil {
			broker = json.RawMessage(`{"type":"object"}`)
		}
		wrapper := append([]byte(`{"type":"object","additionalProperties":false,"required":["broker_result","satgate_receipt"],"properties":{"broker_result":`), broker...)
		return append(wrapper, `,"satgate_receipt":`+bindingReceiptSchema+`}}`...)
	}}), true
}

const bindingCapability = "io.satgate/receipt-binding"

type bindingOptInKey struct{}

func withBindingOptIn(ctx context.Context) context.Context {
	return context.WithValue(ctx, bindingOptInKey{}, true)
}

func bindingInitOptIn(req *Request) bool {
	return req.Method == MethodInitialize && bindingOptInAt(req.Params, "capabilities", "experimental", bindingCapability, "version")
}

func bindingRequestOptIn(ctx context.Context, req *Request) bool {
	if optedIn, _ := ctx.Value(bindingOptInKey{}).(bool); optedIn {
		return true
	}
	return bindingOptInAt(req.Params, "_meta", bindingCapability, "version")
}

// BindingOptedIn reports whether the request or its session opted in to the typed
// receipt handle. It gives the same answer the proxy uses before adding the
// handle. It reads only the request params and the context's opt-in flag, reads
// no token, and logs nothing. A nil request returns false; a nil context is
// treated as context.Background().
func BindingOptedIn(ctx context.Context, req *Request) bool {
	if req == nil {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return bindingRequestOptIn(ctx, req)
}

func bindingOptInAt(raw json.RawMessage, path ...string) bool {
	for _, key := range path {
		var next json.RawMessage
		if _, ok := walkBindingObject(raw, func(name string, start, end int) {
			if name == key {
				next = raw[start:end]
			}
		}); !ok {
			return false
		}
		raw = next
	}
	if len(raw) == 0 || raw[0] < '0' || raw[0] > '9' {
		return false
	}
	// Compare numerically without rounding a nearby decimal to 1.
	digits, exponent := string(raw), 0
	if i := strings.IndexAny(digits, "eE"); i >= 0 {
		var err error
		exponent, err = strconv.Atoi(digits[i+1:])
		if err != nil {
			return false
		}
		digits = digits[:i]
	}
	fractional := 0
	if i := strings.IndexByte(digits, '.'); i >= 0 {
		fractional = len(digits) - i - 1
		digits = digits[:i] + digits[i+1:]
	}
	digits = strings.TrimLeft(digits, "0")
	return strings.TrimRight(digits, "0") == "1" && exponent == fractional-(len(digits)-1)
}

// Offsets refer to the original byte slice, including for nested values.
func walkBindingObject(raw json.RawMessage, visit func(string, int, int)) (int, bool) {
	d := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := d.Token(); err != nil || tok != json.Delim('{') {
		return 0, false
	}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return 0, false
		}
		var size bindingJSONSize
		if err := d.Decode(&size); err != nil {
			return 0, false
		}
		end := int(d.InputOffset())
		visit(key.(string), end-int(size), end)
	}
	if tok, err := d.Token(); err != nil || tok != json.Delim('}') {
		return 0, false
	}
	end := int(d.InputOffset())
	return end - 1, len(bytes.Trim(raw[end:], " \t\r\n")) == 0
}

type bindingJSONSize int

func (n *bindingJSONSize) UnmarshalJSON(raw []byte) error {
	*n = bindingJSONSize(len(raw))
	return nil
}

type bindingJSONField struct {
	key        string
	start, end int
}

type bindingJSONObject struct {
	raw    json.RawMessage
	close  int
	fields []bindingJSONField
}

func parseBindingObject(raw json.RawMessage) (bindingJSONObject, bool) {
	obj := bindingJSONObject{raw: raw}
	close, ok := walkBindingObject(raw, func(key string, start, end int) {
		obj.fields = append(obj.fields, bindingJSONField{key, start, end})
	})
	if !ok {
		return bindingJSONObject{}, false
	}
	obj.close = close
	return obj, true
}

func (o bindingJSONObject) field(key string) json.RawMessage {
	for i := len(o.fields) - 1; i >= 0; i-- {
		f := o.fields[i]
		if f.key == key {
			return o.raw[f.start:f.end]
		}
	}
	return nil
}

type bindingJSONEdit struct {
	key   string
	value func(json.RawMessage) []byte
}

func (o bindingJSONObject) with(edits ...bindingJSONEdit) json.RawMessage {
	var out []byte
	seen := make([]bool, len(edits))
	last := 0
	for _, field := range o.fields {
		for i, edit := range edits {
			if field.key == edit.key {
				out = append(out, o.raw[last:field.start]...)
				out = append(out, edit.value(o.raw[field.start:field.end])...)
				last = field.end
				seen[i] = true
				break
			}
		}
	}
	out = append(out, o.raw[last:o.close]...)
	nonempty := len(o.fields) != 0
	for i, edit := range edits {
		if seen[i] {
			continue
		}
		if nonempty {
			out = append(out, ',')
		}
		out = strconv.AppendQuote(out, edit.key)
		out = append(out, ':')
		out = append(out, edit.value(nil)...)
		nonempty = true
	}
	return append(out, o.raw[o.close:]...)
}
