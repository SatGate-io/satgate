package mcpserver

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// A zero-credit helper's own macaroon signature is the one id the gateway's
// HTTP check reads that names this helper and not its parent. Delegate hands
// it to the event consumer through PersistChild, before the key is returned,
// with the option on or off. The token bytes and caveat list do not change
// (TestZeroCreditChildWithTheOptionOffIsMintedAsProdMintsIt).
func TestZeroCreditHelperSignatureIsHandedToThePersisterBeforeTheKey(t *testing.T) {
	for _, on := range []bool{false, true} {
		ctx := context.Background()
		svc, d, _, parent := helperIDRig(t)
		d.EnableHelperRevocationID(on)
		pe := &persistingEvents{}
		d.SetEventPublisher(pe)
		res, err := d.Delegate(ctx, parent, &DelegateParams{Label: "w", Scope: "api:read"})
		if err != nil || res == nil {
			t.Fatalf("on=%v: res=%v err=%v", on, res, err)
		}
		if len(pe.persisted) != 1 {
			t.Fatalf("on=%v: persisted %d events, want 1", on, len(pe.persisted))
		}
		got, _ := pe.persisted[0].Data[EventChildSignature].(string)
		mac, err := svc.Decode(res.Token)
		if err != nil {
			t.Fatal(err)
		}
		if got == "" || got != mac.Signature {
			t.Fatalf("on=%v: persister got a signature that is not the key's own (len %d)", on, len(got))
		}
		if b, err := hex.DecodeString(got); err != nil || len(b) != 32 || len(got) != 64 {
			t.Fatalf("on=%v: signature is not 64 hex characters", on)
		}
		// The parent's signature is not the helper's.
		pm, _ := svc.Decode(parent.RawToken)
		if got == pm.Signature {
			t.Fatalf("on=%v: persister was given the parent's signature", on)
		}
	}
}

// The signature rides only the persisted copy of the event. The published
// event (what Redis, the audit trail, webhooks and the receipt pipeline see)
// never carries it, and neither does any log line of a helper mint.
func TestHelperSignatureIsNeverPublishedOrLogged(t *testing.T) {
	for _, on := range []bool{false, true} {
		ctx := context.Background()
		svc, d, _, parent := helperIDRig(t)
		d.EnableHelperRevocationID(on)
		pe := &persistingEvents{}
		d.SetEventPublisher(pe)
		var logs bytes.Buffer
		old := log.Logger
		oldLevel := zerolog.GlobalLevel()
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		log.Logger = zerolog.New(&logs).Level(zerolog.TraceLevel)
		res, err := d.Delegate(ctx, parent, &DelegateParams{Label: "w", Scope: "api:read"})
		log.Logger = old
		zerolog.SetGlobalLevel(oldLevel)
		if err != nil {
			t.Fatal(err)
		}
		mac, _ := svc.Decode(res.Token)
		sig := mac.Signature
		if len(pe.events) != 1 {
			t.Fatalf("on=%v: published %d events, want 1", on, len(pe.events))
		}
		if _, there := pe.events[0].Data[EventChildSignature]; there {
			t.Fatalf("on=%v: the published event carries the helper's signature", on)
		}
		for k, v := range pe.events[0].Data {
			if s, ok := v.(string); ok && strings.Contains(s, sig) {
				t.Fatalf("on=%v: published event field %q holds the signature", on, k)
			}
		}
		if logs.Len() == 0 {
			t.Fatalf("on=%v: captured no log output, so this test proves nothing", on)
		}
		if strings.Contains(logs.String(), sig) || strings.Contains(logs.String(), res.Token) {
			t.Fatalf("on=%v: a log line of the helper mint carries the signature or the key: %s", on, logs.String())
		}
	}
}

// The persisted copy is a copy: adding the signature to it must not leak into
// the event Delegate publishes afterwards (the maps are not shared).
func TestPersistedEventIsACopyOfThePublishedOne(t *testing.T) {
	ctx := context.Background()
	_, d, _, parent := helperIDRig(t)
	pe := &persistingEvents{}
	d.SetEventPublisher(pe)
	if _, err := d.Delegate(ctx, parent, &DelegateParams{Label: "w"}); err != nil {
		t.Fatal(err)
	}
	if _, there := pe.persisted[0].Data[EventChildSignature]; !there {
		t.Fatal("persisted copy lacks the signature")
	}
	if _, there := pe.events[0].Data[EventChildSignature]; there {
		t.Fatal("published event shares the persisted copy's map")
	}
}

// Storing is part of issuing for the option-off helper too: no record, no key,
// and the client sees the fixed sentence.
func TestOptionOffHelperWhoseRecordFailsGetsNoKey(t *testing.T) {
	ctx := context.Background()
	_, d, _, parent := helperIDRig(t)
	d.EnableHelperRevocationID(false)
	pe := &persistingEvents{err: errors.New("pq: could not store secret-row-text")}
	d.SetEventPublisher(pe)
	var logs bytes.Buffer
	old := log.Logger
	log.Logger = zerolog.New(&logs)
	res, err := d.Delegate(ctx, parent, &DelegateParams{Label: "w", Scope: "api:read"})
	log.Logger = old
	if err == nil || res != nil {
		t.Fatalf("res=%v err=%v, want no key", res, err)
	}
	if got := delegationClientMessage(err); got != ErrChildTokenFailed.Error() {
		t.Fatalf("client text = %q", got)
	}
	if strings.Contains(err.Error(), "secret-row-text") || strings.Contains(logs.String(), "secret-row-text") {
		t.Fatal("the store error text escaped")
	}
	if len(pe.events) != 0 {
		t.Fatalf("a refused delegation published %d events", len(pe.events))
	}
}

// A publisher that cannot record (not a ChildPersister, as in the OSS build
// with its no-op publisher) is unchanged: the key is returned.
func TestOptionOffHelperWithAPublisherThatCannotRecordStillGetsItsKey(t *testing.T) {
	ctx := context.Background()
	_, d, ev, parent := helperIDRig(t)
	d.EnableHelperRevocationID(false)
	d.SetEventPublisher(ev)
	if res, err := d.Delegate(ctx, parent, &DelegateParams{Label: "w"}); err != nil || res == nil {
		t.Fatalf("res=%v err=%v", res, err)
	}
}

// An own-credit child is named by its budget id: the persister is not asked
// and the signature is not handed over.
func TestOwnCreditChildSignatureIsNotHandedOver(t *testing.T) {
	ctx := context.Background()
	_, d, _, parent := helperIDRig(t)
	pe := &persistingEvents{}
	d.SetEventPublisher(pe)
	if _, err := d.Delegate(ctx, parent, &DelegateParams{Label: "funded", Budget: 10}); err != nil {
		t.Fatal(err)
	}
	if len(pe.persisted) != 0 {
		t.Fatalf("persister was asked for an own-credit child")
	}
	if _, there := pe.events[0].Data[EventChildSignature]; there {
		t.Fatal("event carries a signature")
	}
}
