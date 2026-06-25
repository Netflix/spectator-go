package meter

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"github.com/cespare/xxhash/v2"
)

const vectorFile = "testdata/distinct_count_sketch_test_vectors.txt"

// hashOfVector computes the xxHash64 of an encoding-vector input the same way the sketch does for
// each type: long as 8-byte little-endian, str as UTF-8 bytes, bytes as raw bytes. This is an
// independent check that the Go hashing matches the hash column produced by spectator-java.
func hashOfVector(t *testing.T, typ, input string) uint64 {
	switch typ {
	case "long":
		v, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			t.Fatalf("bad long input %q: %v", input, err)
		}
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(v))
		return xxhash.Sum64(buf[:])
	case "str":
		return xxhash.Sum64String(unescapeVector(input))
	case "bytes":
		b, err := hex.DecodeString(input)
		if err != nil {
			t.Fatalf("bad bytes input %q: %v", input, err)
		}
		return xxhash.Sum64(b)
	default:
		t.Fatalf("unknown vector type %q", typ)
		return 0
	}
}

// recordVector drives a vector input through the production Record* method for its type.
func recordVector(t *testing.T, d *DistinctCountSketch, typ, input string) {
	switch typ {
	case "long":
		v, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			t.Fatalf("bad long input %q: %v", input, err)
		}
		d.RecordInt64(v)
	case "str":
		d.RecordString(unescapeVector(input))
	case "bytes":
		b, err := hex.DecodeString(input)
		if err != nil {
			t.Fatalf("bad bytes input %q: %v", input, err)
		}
		d.RecordBytes(b)
	default:
		t.Fatalf("unknown vector type %q", typ)
	}
}

// unescapeVector reverses the escaping used for str values in the vector file (\\, \t, \n, \r).
func unescapeVector(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case '\\':
				b.WriteByte('\\')
			case 't':
				b.WriteByte('\t')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// TestDistinctCountSketch_HashVectors verifies that the Go xxHash64 of each encoding-vector input
// matches the hash produced by the spectator-java reference implementation. This is what the 's'
// (precomputed hash) line type sends to SpectatorD, so matching hashes is what lets Go-recorded
// sketches merge with sketches from the other clients.
func TestDistinctCountSketch_HashVectors(t *testing.T) {
	f, err := os.Open(vectorFile)
	if err != nil {
		t.Fatalf("cannot open vector file: %v", err)
	}
	defer f.Close()

	inSection := false
	checked := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = line == "[encoding]"
			continue
		}
		if !inSection {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 5 {
			t.Fatalf("malformed encoding line: %q", line)
		}
		typ, input, hashHex := fields[0], fields[1], fields[2]
		expected, err := strconv.ParseUint(hashHex, 16, 64)
		if err != nil {
			t.Fatalf("bad hash %q: %v", hashHex, err)
		}
		if got := hashOfVector(t, typ, input); got != expected {
			t.Errorf("hash mismatch for %s %q: got %016x, want %016x", typ, input, got, expected)
		}

		// Drive the input through the production Record* methods and confirm the emitted line
		// carries the hash from the vector file, tying the shipped code to the cross-language
		// reference (covers negatives, min int64, multibyte strings, and empty values).
		w := writer.MemoryWriter{}
		d := NewDistinctCountSketch(NewId("test", nil), &w)
		recordVector(t, d, typ, input)
		wantLine := "s:test:" + strconv.FormatUint(expected, 10)
		if lines := w.Lines(); len(lines) != 1 || lines[0] != wantLine {
			t.Errorf("line mismatch for %s %q: got %v, want %q", typ, input, lines, wantLine)
		}
		checked++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading vector file: %v", err)
	}
	if checked == 0 {
		t.Fatal("no encoding vectors were checked")
	}
}

func TestDistinctCountSketch_RecordString(t *testing.T) {
	w := writer.MemoryWriter{}
	d := NewDistinctCountSketch(NewId("dcs", nil), &w)
	// "a" hashes to 0xd24ec4f1a98c6e5b == 15154266338359012955 (see the vector file).
	d.RecordString("a")

	expected := "s:dcs:15154266338359012955"
	if lines := w.Lines(); len(lines) != 1 || lines[0] != expected {
		t.Errorf("expected %q, got %v", expected, lines)
	}
}

func TestDistinctCountSketch_RecordStringMatchesBytes(t *testing.T) {
	// RecordString(s) and RecordBytes([]byte(s)) must agree, since both hash the same bytes.
	ws := writer.MemoryWriter{}
	wb := writer.MemoryWriter{}
	NewDistinctCountSketch(NewId("dcs", nil), &ws).RecordString("user-12345")
	NewDistinctCountSketch(NewId("dcs", nil), &wb).RecordBytes([]byte("user-12345"))
	if ws.Lines()[0] != wb.Lines()[0] {
		t.Errorf("RecordString and RecordBytes disagree: %q vs %q", ws.Lines()[0], wb.Lines()[0])
	}
}
