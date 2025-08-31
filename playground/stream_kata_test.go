package playground

import (
	"errors"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudwego/eino/schema"
)

// TestPipeCorrect shows the canonical pattern: writer Send -> Close; reader Recv until io.EOF -> Close.
func TestPipeCorrect(t *testing.T) {
	sr, sw := schema.Pipe[int](2)
	defer sr.Close()

	// producer
	go func() {
		for i := 0; i < 5; i++ {
			closed := sw.Send(i, nil)
			if closed {
				return
			}
		}
		sw.Close()
	}()

	// consumer
	var got []int
	for {
		v, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err)
		got = append(got, v)
	}

	assert.Equal(t, []int{0, 1, 2, 3, 4}, got)
}

// TestReaderEarlyExitMustClose demonstrates that if reader exits early, it must call sr.Close()
// otherwise writer may block on Send when buffer is full.
func TestReaderEarlyExitMustClose(t *testing.T) {
	sr, sw := schema.Pipe[int](1)

	var sent uint32
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { // writer: try to send many items; will block without reader Close
		defer wg.Done()
		for i := 0; i < 100; i++ {
			if sw.Send(i, nil) { // closed by reader.Close()
				break
			}
			atomic.AddUint32(&sent, 1)
		}
		close(done)
	}()

	// reader: reads only one item, then exits WITHOUT closing initially
	_, err := sr.Recv()
	assert.NoError(t, err)

	// give writer time to progress; with cap=1 and no further reads, writer should not finish
	time.Sleep(50 * time.Millisecond)
	assert.NotEqual(t, uint32(100), atomic.LoadUint32(&sent), "writer should not be able to finish sending without reader Close")

	// now do the right thing: close reader to notify writer to stop
	sr.Close()

	// writer should finish quickly after reader closes (done is closed in writer goroutine)
	select {
	case <-done:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatal("writer didn't finish after reader Close()")
	}
}

// TestWriterMissingCloseMustClose shows that writer must Close() so that reader can observe io.EOF.
func TestWriterMissingCloseMustClose(t *testing.T) {
	sr, sw := schema.Pipe[int](0)
	defer sr.Close()

	// writer sends some but forgets to close
	go func() {
		_ = sw.Send(1, nil)
		_ = sw.Send(2, nil)
		// sw.Close() intentionally omitted here
		// later we close from test to demonstrate unblocking
	}()

	// reader drains existing items
	got := make([]int, 0, 2)
	for i := 0; i < 2; i++ {
		v, err := sr.Recv()
		assert.NoError(t, err)
		got = append(got, v)
	}

	// next Recv would block forever without writer Close(); guard with timeout.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = sr.Recv() // would block until we Close writer
	}()

	select {
	case <-done:
		t.Fatal("unexpected: Recv finished without EOF; writer probably closed prematurely")
	case <-time.After(50 * time.Millisecond):
		// expected blocked
	}

	// now close writer to deliver EOF
	sw.Close()

	select {
	case <-done:
		// Recv should return io.EOF now
	case <-time.After(500 * time.Millisecond):
		t.Fatal("reader didn't finish after writer Close()")
	}

	assert.Equal(t, []int{1, 2}, got)
}

// TestCopyTwoSubscribers verifies Copy creates independent readers and original should no longer be used.
func TestCopyTwoSubscribers(t *testing.T) {
	sr, sw := schema.Pipe[int](2)

	// produce 3 values then close
	go func() {
		for i := 1; i <= 3; i++ {
			if sw.Send(i, nil) {
				return
			}
		}
		sw.Close()
	}()

	// copy into two readers; sr itself should be considered unusable afterwards
	copies := sr.Copy(2)
	r1, r2 := copies[0], copies[1]
	defer r1.Close()
	defer r2.Close()

	// each reader should receive the same sequence independently
	var a, b []int
	for {
		v, err := r1.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err)
		a = append(a, v)
	}
	for {
		v, err := r2.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err)
		b = append(b, v)
	}

	assert.Equal(t, []int{1, 2, 3}, a)
	assert.Equal(t, []int{1, 2, 3}, b)

	// Do not Close the original sr after Copy; parent will manage underlying closure when children close.
}

// TestConvertFilterAndError shows StreamReaderWithConvert can filter values via ErrNoValue and propagate other errors.
func TestConvertFilterAndError(t *testing.T) {
	sr, sw := schema.Pipe[int](2)

	// produce: 1, 0 (filtered), 2, 99 (error), 3
	go func() {
		for _, v := range []int{1, 0, 2, 99, 3} {
			if sw.Send(v, nil) {
				return
			}
		}
		sw.Close()
	}()

	conv := func(i int) (string, error) {
		if i == 0 {
			return "", schema.ErrNoValue // filtered out
		}
		if i == 99 {
			return "", assert.AnError // arbitrary error for test
		}
		return "v_" + itoa(i), nil
	}

	out := schema.StreamReaderWithConvert[int, string](sr, conv)
	defer out.Close()

	vals := []string{}
	var gotErr error
	for {
		v, err := out.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			gotErr = err
			continue
		}
		vals = append(vals, v)
	}

	// 0 is filtered, 99 raises error, others pass
	assert.Equal(t, []string{"v_1", "v_2", "v_3"}, vals)
	assert.Error(t, gotErr)

	// close the converted reader; the underlying reader will be closed by it
}

// minimal integer to string without fmt to avoid extra import
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// Teaching: child1 reads first then child0, both must see same first element.
func TestCopyFirstElementChild1FirstPlayground(t *testing.T) {
	sr, sw := schema.Pipe[int](0)

	go func() {
		_ = sw.Send(1, nil)
		time.Sleep(10 * time.Millisecond)
		_ = sw.Send(2, nil)
		sw.Close()
	}()

	copies := sr.Copy(2)
	c0, c1 := copies[0], copies[1]
	defer c0.Close()
	defer c1.Close()

	v1a, err := c1.Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v1a)

	v0a, err := c0.Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v0a)

	v1b, err := c1.Recv()
	assert.NoError(t, err)
	assert.Equal(t, 2, v1b)

	v0b, err := c0.Recv()
	assert.NoError(t, err)
	assert.Equal(t, 2, v0b)
}

// Teaching: one child closes early, the other continues to EOF.
func TestCopyOneChildCloseEarlyPlayground(t *testing.T) {
	sr, sw := schema.Pipe[int](8)

	go func() {
		for i := 0; i < 100; i++ {
			if i%13 == 0 {
				time.Sleep(time.Microsecond * time.Duration(rand.Intn(200)))
			}
			if sw.Send(i, nil) {
				return
			}
		}
		sw.Close()
	}()

	copies := sr.Copy(2)
	c0, c1 := copies[0], copies[1]
	defer c1.Close()

	// c0 reads one then closes
	v0, err := c0.Recv()
	assert.NoError(t, err)
	assert.Equal(t, 0, v0)
	c0.Close()

	// c1 continues to EOF
	expect := 0
	for {
		v, err := c1.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err)
		if expect == 0 {
			// first element was already read by c0; c1 still must see it
			assert.Equal(t, 0, v)
		} else {
			assert.Equal(t, expect, v)
		}
		expect++
	}
}
