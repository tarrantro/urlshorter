// Simple Twitter snowflake generator and parser.
package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var (
	// Epoch is set to the epoch of Feb 02 2024 07:10:34 UTC in milliseconds
	Epoch int64 = 1706857834902

	// TimeBits holds the number of bits to use for TimeStamp milliseconds since the Epoch
	TimeBits uint8 = 41

	// NodeBits holds the number of bits to use for Node
	// Remember, you have a total 22 bits to share between Node/Step
	NodeBits uint8 = 10

	// StepBits holds the number of bits to use for Step
	// Remember, you have a total 22 bits to share between Node/Step
	StepBits uint8 = 12

	timeShift       = NodeBits + StepBits
	nodeShift       = StepBits
)

const encodeBase62Map = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var decodeBase62Map [256]byte

// A JSONSyntaxError is returned from UnmarshalJSON if an invalid ID is provided.
type JSONSyntaxError struct{ original []byte }

func (j JSONSyntaxError) Error() string {
	return fmt.Sprintf("invalid snowflake ID %q", string(j.original))
}

// ErrInvalidBase62 is returned by ParseBase62 when given an invalid []byte
var ErrInvalidBase62 = errors.New("invalid base62")

// Create maps for decoding Base62.
func init() {

	for i := 0; i < len(encodeBase62Map); i++ {
		decodeBase62Map[i] = 0xFF
	}

	for i := 0; i < len(encodeBase62Map); i++ {
		decodeBase62Map[encodeBase62Map[i]] = byte(i)
	}
}

// A Node struct holds the basic information needed for a snowflake generator
// node
type Node struct {
	mu   sync.Mutex
	time int64
	node int
	step int
}

// An ID is a custom type used for a snowflake ID.  This is used so we can
// attach methods onto the ID.
type ID int64

// NewNode returns a new snowflake node that can be used to generate snowflake
// IDs
func NewNode(nodes ...int) (*Node, error) {

	var node int

	if len(nodes) > 0 {
		node = nodes[0]
	} else {
		node = rand.Intn(32768)
	}

	if NodeBits+StepBits > 22 {
		return nil, errors.New("Remember, you have a total 22 bits to share between Node/Step")
	}

	n := Node{}
	nodeMask := ^(^0 << NodeBits)
	// Node ID(Machine ID) must be in 10 bits(0-1023)
	n.node = node & nodeMask
	return &n, nil
}

// Generate creates and returns a unique snowflake ID
// To help guarantee uniqueness
// - Make sure your system is keeping accurate system time
// - Make sure you never have multiple nodes running with the same node ID
func (n *Node) Generate() (ID, error) {

	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Since(time.Unix(0, Epoch)).Milliseconds()

	// If the clock turns back, increment the machine seq id till 0
	if now <= n.time {
		stepMask := ^(^0 << StepBits)
		n.step = (n.step + 1) & stepMask
		if n.step == 0 {
			// Try to wait 3s to see if the clock back to normal
			count := 3
			for now <= n.time {
				time.Sleep(1 * time.Second)
				now = time.Since(time.Unix(0, Epoch)).Milliseconds()
				if count <= 0 {
					return -1, errors.New("Error in epoch, please check if your local clock ever turn back.")
				}
				count -= 1
			}
		}
	} else {
		n.step = 0
	}

	n.time = now
	r := ID((now)<<timeShift |
		int64((n.node << nodeShift)) |
		int64((n.step)),
	)

	return r, nil
}

// Int64 returns an int64 of the snowflake ID
func (f ID) Int64() int64 {
	return int64(f)
}

// ParseInt64 converts an int64 into a snowflake ID
func ParseInt64(id int64) ID {
	return ID(id)
}

// String returns a string of the snowflake ID
func (f ID) String() string {
	return strconv.FormatInt(int64(f), 10)
}

// ParseString converts a string into a snowflake ID
func ParseString(id string) (ID, error) {
	i, err := strconv.ParseInt(id, 10, 64)
	return ID(i), err

}

// Base62 uses the z-base-62 character set but encodes and decodes similar
func (f ID) Base62(digits ...int) string {
	d := 11
	if len(digits) > 0 {
		if digits[0] > 0 {
			d = digits[0]
		}
	}

	b := make([]byte, d)
	for i := d - 1; i >= 0; i-- {
		if f < 62 {
			b[i] = encodeBase62Map[f]
		} else {
			b[i] = encodeBase62Map[f%62]
		}
		f /= 62
	}

	return string(b)
}

// Bytes returns a byte slice of the snowflake ID
func (f ID) Bytes() []byte {
	return []byte(f.String())
}

// ParseBytes converts a byte slice into a snowflake ID
func ParseBytes(id []byte) (ID, error) {
	i, err := strconv.ParseInt(string(id), 10, 64)
	return ID(i), err
}

// IntBytes returns an array of bytes of the snowflake ID, encoded as a
// big endian integer.
func (f ID) IntBytes() [8]byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(f))
	return b
}

// ParseIntBytes converts an array of bytes encoded as big endian integer as
// a snowflake ID
func ParseIntBytes(id [8]byte) ID {
	return ID(int64(binary.BigEndian.Uint64(id[:])))
}

// MarshalJSON returns a json byte array string of the snowflake ID.
func (f ID) MarshalJSON() ([]byte, error) {
	buff := make([]byte, 0, 22)
	buff = append(buff, '"')
	buff = strconv.AppendInt(buff, int64(f), 10)
	buff = append(buff, '"')
	return buff, nil
}

// UnmarshalJSON converts a json byte array of a snowflake ID into an ID type.
func (f *ID) UnmarshalJSON(b []byte) error {
	if len(b) < 3 || b[0] != '"' || b[len(b)-1] != '"' {
		return JSONSyntaxError{b}
	}

	i, err := strconv.ParseInt(string(b[1:len(b)-1]), 10, 64)
	if err != nil {
		return err
	}

	*f = ID(i)
	return nil
}
