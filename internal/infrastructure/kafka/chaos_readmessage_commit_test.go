package kafka

// ---------------------------------------------------------------------------
// What this file is testing
//
// Consumer.Start (consumer.go) does:
//
//	message, err := c.reader.ReadMessage(ctx)
//	...
//	if err := c.handler(ctx, message); err != nil {
//	    // ... publish to DLQ ...
//	}
//
// kafka-go's own documentation for Reader.ReadMessage says plainly:
// "ReadMessage automatically commits offsets when using consumer groups"
// and recommends FetchMessage + CommitMessages instead "if more
// fine-grained control of when offsets are committed is required."
//
// That's not a footnote - it means the offset for `message` is already
// committed to the broker by the time ReadMessage returns, which is BEFORE
// c.handler runs at all. For TradeConsumer, c.handler is what calls
// settlementservice.Settle - the thing that writes the trade row and moves
// money between wallets.
//
// So: the trade has already matched in the engine (it's on the WAL, the
// order book has been updated), Kafka considers the trade-executed event
// fully consumed the instant ReadMessage returns - and only THEN does
// settlement even begin. If the process dies anywhere between those two
// points - OOM-kill, panic, a bad deploy, the pod getting rescheduled -
// there is no redelivery. The DLQ path in Consumer.Start only helps when
// the handler is alive long enough to return an error; it can't help with
// a crash that happens while the handler is still running, because the
// offset was already gone before the handler was ever called.
//
// Net effect: a process crash during trade settlement can silently and
// permanently lose that trade. No trade row, no wallet movement, no
// failed_settlements record, and no way for Kafka to ever hand the
// message back - the exact opposite of what "at-least-once" is supposed
// to guarantee.
//
// This test doesn't go through TradeConsumer or a real crash (an
// unrecovered panic would kill the whole test binary, not just simulate
// one). It isolates the actual mechanism at the kafka.Reader level, which
// is both simpler and more convincing: read one message and close that
// reader immediately, doing zero processing - then open a brand new
// reader in the same consumer group and confirm the message is gone for
// good, not redelivered.
//
// Requires a real broker at localhost:9092 (see deployments/compose) -
// same as the existing TestConsumer/TestProducer tests in this package.
// ---------------------------------------------------------------------------

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/stretchr/testify/require"
)

func TestChaos_ReadMessageCommitsBeforeProcessing_LosesMessageOnCrash(t *testing.T) {
	topic := "velocity-chaos-test-" + uuid.NewString()[:8]
	groupID := "velocity-chaos-group-" + uuid.NewString()[:8]
	brokers := []string{"localhost:9092"}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// This broker has topic auto-creation disabled, so the topic must be
	// provisioned explicitly - same helper already used by
	// TestEnsureTopics in this package.
	require.NoError(t, EnsureTopics(brokers, TopicConfig{
		Name:              topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}))

	defer func() {
		conn, err := kafka.Dial("tcp", brokers[0])
		if err != nil {
			t.Logf("cleanup: failed to dial kafka: %v", err)
			return
		}
		defer conn.Close()
		if err := conn.DeleteTopics(topic); err != nil {
			t.Logf("cleanup: failed to delete test topic %s: %v", topic, err)
		}
	}()

	producer := NewProducer(brokers, topic)
	defer producer.Close()

	require.NoError(t, producer.Publish(ctx, "trade-X", map[string]any{
		"trade_id": "X",
		"note":     "this is the trade whose settlement will never run",
	}))

	// ------------------------------------------------------------------
	// Step 1: a consumer picks up trade X, then "crashes" - simulated by
	// closing the reader immediately, without calling any handler. If
	// ReadMessage's commit genuinely happens before processing, this is
	// enough on its own to lose the message - no panic or os.Exit needed
	// to prove it.
	// ------------------------------------------------------------------

	reader1 := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})

	msg1, err := reader1.ReadMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, "trade-X", string(msg1.Key))

	require.NoError(t, reader1.Close())
	// Nothing else happens with msg1. This is the crash: settlement for
	// trade X never runs, no trade row, no wallet update, no
	// failed_settlements record - and per the assertion below, no second
	// chance either.

	// ------------------------------------------------------------------
	// Step 2: publish a second, distinct trade, so we can tell "the
	// group is broken" apart from "trade X specifically was consumed."
	// ------------------------------------------------------------------

	require.NoError(t, producer.Publish(ctx, "trade-Y", map[string]any{
		"trade_id": "Y",
		"note":     "published after the crash, should still arrive fine",
	}))

	// ------------------------------------------------------------------
	// Step 3: a fresh reader - standing in for the consumer process
	// restarting - joins the SAME consumer group and reads the next
	// message it's handed.
	// ------------------------------------------------------------------

	reader2 := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})
	defer reader2.Close()

	msg2, err := reader2.ReadMessage(ctx)
	require.NoError(t, err)

	require.Equalf(t, "trade-Y", string(msg2.Key),
		"BUG: expected the restarted consumer to receive trade-Y (proving "+
			"trade-X was never redelivered), but got key=%q instead. If "+
			"this is \"trade-X\", ReadMessage's auto-commit is NOT the "+
			"cause of message loss here and this finding needs revisiting. "+
			"If it's neither, the topic/group setup itself is the problem.",
		string(msg2.Key))

	// The group's committed offset is now past both messages, and no
	// third message exists - so a bounded read confirms trade-X is truly
	// gone, not just delayed behind trade-Y.
	shortCtx, shortCancel := context.WithTimeout(ctx, 3*time.Second)
	defer shortCancel()

	_, err = reader2.ReadMessage(shortCtx)
	require.Error(t, err,
		"expected no further messages (specifically, trade-X should never "+
			"reappear) - got one instead, which would itself be worth "+
			"investigating")
}

// TestChaos_ConsumerRedeliversAfterAbandonedHandler proves the fix in
// consumer.go: a message whose handler never got the chance to finish (or
// return an error) - standing in for a process crash mid-settlement - must
// be redelivered to the next consumer in the group, not silently dropped.
// This exercises the real Consumer type, not the raw kafka.Reader, since
// that's what production code actually runs.
func TestChaos_ConsumerRedeliversAfterAbandonedHandler(t *testing.T) {
	topic := "velocity-chaos-test-" + uuid.NewString()[:8]
	groupID := "velocity-chaos-group-" + uuid.NewString()[:8]
	brokers := []string{"localhost:9092"}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, EnsureTopics(brokers, TopicConfig{
		Name:              topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}))

	defer func() {
		conn, err := kafka.Dial("tcp", brokers[0])
		if err != nil {
			t.Logf("cleanup: failed to dial kafka: %v", err)
			return
		}
		defer conn.Close()
		if err := conn.DeleteTopics(topic); err != nil {
			t.Logf("cleanup: failed to delete test topic %s: %v", topic, err)
		}
	}()

	producer := NewProducer(brokers, topic)
	defer producer.Close()

	require.NoError(t, producer.Publish(ctx, "trade-Z", map[string]any{
		"trade_id": "Z",
	}))

	// consumer1's handler stands in for a settlement call that never
	// gets to finish - the process "crashes" while it's in progress.
	// handlerStarted fires the instant FetchMessage hands it the
	// message, proving the message really was picked up before the
	// simulated crash, not merely never fetched.
	handlerStarted := make(chan struct{})
	neverCompletes := make(chan struct{}) // deliberately never closed

	consumer1 := NewConsumer(brokers, topic, groupID,
		func(ctx context.Context, message kafka.Message) error {
			close(handlerStarted)
			<-neverCompletes // simulates a crash: this call never returns
			return nil
		},
		nil,
	)

	// consumer1.Start runs in its own goroutine and hangs in the handler
	// for the rest of the test - that's the point. The handler goroutine
	// itself is abandoned (leaked deliberately), exactly as a real crash
	// would leave it: never returning, never committing.
	go func() { _ = consumer1.Start(ctx) }()

	select {
	case <-handlerStarted:
	case <-ctx.Done():
		t.Fatal("timed out waiting for consumer1 to pick up trade-Z")
	}

	// Close consumer1's reader now, forcing an immediate, clean
	// LeaveGroup. This is NOT the same as the handler completing - the
	// handler goroutine above is still blocked on neverCompletes and
	// will remain so for the rest of the test, so trade-Z's offset was
	// never committed. Closing the reader only releases consumer1's
	// partition assignment promptly, so the redelivery this test checks
	// for isn't gated behind Kafka's session-timeout for a member that
	// stopped heartbeating - that would test rebalance timing, not the
	// fix.
	require.NoError(t, consumer1.Close())

	received := make(chan kafka.Message, 1)

	consumer2 := NewConsumer(brokers, topic, groupID,
		func(ctx context.Context, message kafka.Message) error {
			received <- message
			return nil
		},
		nil,
	)
	defer consumer2.Close()

	go func() {
		if err := consumer2.Start(ctx); err != nil {
			t.Logf("consumer2 stopped: %v", err)
		}
	}()

	select {
	case message := <-received:
		require.Equal(t, "trade-Z", string(message.Key),
			"redelivered message should be trade-Z, not something else")

	case <-ctx.Done():
		t.Fatal(
			"BUG: trade-Z was never redelivered to a fresh consumer in " +
				"the same group after the handler that first received it " +
				"got stuck - this is the message-loss window the fix in " +
				"consumer.go (FetchMessage + commit-after-success) exists " +
				"to close",
		)
	}
}