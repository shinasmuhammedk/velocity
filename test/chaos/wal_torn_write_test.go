package chaos_test

// ---------------------------------------------------------------------------
// What this file is testing
//
// wal.Writer.Write() does, in order: serialize -> file.Write() -> file.Sync()
// -> advance in-memory sequence. That ordering means the durability promise
// is "if Sync() returned, the record is safe" - it says nothing about what's
// on disk if the process dies *during* the write, before Sync() ever runs.
//
// On a real crash (OOM-kill, power loss, panic, SIGKILL) that's exactly what
// happens: the OS can leave a partially-flushed final line in the file - a
// "torn write". Any durable single-file WAL has to tolerate this on replay;
// a torn trailing record represents a write that never definitely happened,
// so it should be discarded, not treated as corruption.
//
// wal.Reader.ReadAll() (internal/engine/wal/reader.go) does not do this
// today. It scans the file line by line and calls Deserialize on every line
// bufio.Scanner hands back, including a final line with no trailing '\n'.
// A torn line is - by construction - invalid JSON, so Deserialize fails and
// ReadAll returns that error to its caller.
//
// That caller is not just diagnostic code: wal.NewWriter() itself calls
// NewReader(path, ...).ReadAll() at construction time, to recover the last
// persisted sequence. And wal.Manager.Writer() - which is what
// registry.Get() calls on every lazy engine creation - returns nil on that
// error, which registry.Get() then returns as a bare nil *engine.Engine
// with no panic, no log, and no visible indication anything went wrong.
//
// Net effect: a single torn write at the tail of one symbol's WAL file -
// plausible on any hard crash - makes that symbol's engine unable to start
// again, ever, until someone manually edits the WAL file. Every other
// symbol is fine, so this fails quietly and only for the affected symbol,
// which is a worse failure mode than an obvious startup crash would be.
//
// Both tests below assert the behavior a durable WAL is supposed to have
// (torn tail discarded, valid prefix replayed, engine starts normally).
// They are expected to FAIL against the current reader.go - that failure
// *is* the bug report. The fix belongs in wal.Reader.ReadAll(): if
// Deserialize fails on what scanner.Scan() confirms is the last line, log a
// warning and stop reading, rather than propagating the error.
// ---------------------------------------------------------------------------

import (
	"os"
	"path/filepath"
	"testing"

	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"

	"github.com/stretchr/testify/require"
)

// writeWALWithTornTail writes `validCount` well-formed WAL records to
// <dir>/<symbol>.wal, exactly as wal.Writer would, and then appends a
// deliberately truncated JSON line with no trailing newline - simulating a
// process dying mid os.File.Write() on the next record.
func writeWALWithTornTail(t *testing.T, dir, symbol string, validCount int) {
	t.Helper()

	serializer := wal.NewJSONSerializer()
	writer, err := wal.NewWriter(dir, symbol, serializer)
	require.NoError(t, err)

	for i := 1; i <= validCount; i++ {
		event := wal.NewCancelEvent(uint64(i), symbol, int64(1000+i))
		require.NoError(t, writer.Write(event))
	}
	require.NoError(t, writer.Close())

	// Simulate the crash: append a truncated record directly, bypassing
	// Writer entirely, with no trailing '\n' - the exact shape a torn
	// os.File.Write() leaves behind.
	path := filepath.Join(dir, symbol+".wal")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	defer f.Close()

	tornRecord := `{"sequence":` // cut off mid-object, no closing brace, no newline
	_, err = f.WriteString(tornRecord)
	require.NoError(t, err)
}

// TestChaos_TornTrailingWALRecord_IsDiscardedOnReplay asserts that a torn
// final record does not prevent the valid prefix of the WAL from being
// read back.
func TestChaos_TornTrailingWALRecord_IsDiscardedOnReplay(t *testing.T) {
	dir := t.TempDir()
	symbol := "BTCUSDT"

	writeWALWithTornTail(t, dir, symbol, 3)

	reader, err := wal.NewReader(
		filepath.Join(dir, symbol+".wal"),
		wal.NewJSONSerializer(),
	)
	require.NoError(t, err)
	defer reader.Close()

	events, err := reader.ReadAll()
	require.NoError(t, err,
		"a torn trailing record should be discarded, not fail the entire "+
			"replay - see internal/engine/wal/reader.go ReadAll()")

	require.Len(t, events, 3,
		"expected exactly the 3 well-formed records written before the "+
			"simulated crash; the torn 4th record must not appear at all")

	for i, event := range events {
		require.EqualValues(t, i+1, event.Sequence)
	}
}

// TestChaos_TornTrailingWALRecord_MustNotBlockEngineRecovery asserts the
// consequence that actually matters in production: registry.Get() must
// still be able to start an engine for a symbol whose WAL ends in a torn
// write. Today it silently returns nil instead, because wal.NewWriter()
// calls ReadAll() internally and propagates the same error.
func TestChaos_TornTrailingWALRecord_MustNotBlockEngineRecovery(t *testing.T) {
	dir := t.TempDir()
	symbol := "BTCUSDT"

	writeWALWithTornTail(t, dir, symbol, 3)

	walManager := wal.NewManager(dir, wal.NewJSONSerializer())

	r := registry.New(&snapshot.MockWriter{}, walManager)
	defer r.Shutdown()

	e := r.Get(symbol)

	require.NotNil(t, e,
		"registry.Get returned a nil engine for a symbol whose WAL has a "+
			"torn trailing record - a real crash leaves exactly this on "+
			"disk, and every subsequent operation on this symbol (order "+
			"submission, cancellation, market data) will now nil-pointer "+
			"panic or silently no-op, with nothing in the logs pointing "+
			"at the WAL file as the cause")
}

// TestChaos_CorruptionInTheMiddleOfWAL_StillFailsHard is the counterpart to
// the two tests above: it proves the fix is narrowly scoped to a torn
// *trailing* record and does not turn into "ignore any bad line anywhere."
// A malformed line with valid records still following it is not a crash
// artifact - it's real corruption (disk fault, manual edit, a serializer
// bug) - and swallowing that silently would be worse than the original
// bug: it would hide missing committed orders instead of just failing to
// start.
func TestChaos_CorruptionInTheMiddleOfWAL_StillFailsHard(t *testing.T) {
	dir := t.TempDir()
	symbol := "BTCUSDT"

	serializer := wal.NewJSONSerializer()
	writer, err := wal.NewWriter(dir, symbol, serializer)
	require.NoError(t, err)

	require.NoError(t, writer.Write(wal.NewCancelEvent(1, symbol, 1001)))
	require.NoError(t, writer.Close())

	path := filepath.Join(dir, symbol+".wal")

	// Inject a corrupt line, followed by ANOTHER valid line after it -
	// i.e. the corrupt line is not the tail, so this cannot be mistaken
	// for a torn write no matter how that's detected.
	validTail, err := serializer.Serialize(
		wal.NewCancelEvent(2, symbol, 1002),
	)
	require.NoError(t, err)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.WriteString("{this is not valid json at all}\n")
	require.NoError(t, err)
	_, err = f.Write(append(validTail, '\n'))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	writer2, err := wal.NewWriter(dir, symbol, serializer)
	// NewWriter calls ReadAll() internally, so mid-file corruption
	// should surface right here rather than being silently accepted.
	if err == nil {
		defer writer2.Close()
	}
	require.Error(t, err,
		"a malformed record in the middle of the WAL (not the trailing "+
			"record) must still be a hard error - only a torn *trailing* "+
			"write should ever be discarded")
}