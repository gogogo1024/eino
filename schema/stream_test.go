/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package schema

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStream(t *testing.T) {
	s := newStream[int](0)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			closed := s.send(i, nil)
			if closed {
				break
			}
		}
		s.closeSend()
	}()

	i := 0
	for {
		i++
		if i > 5 {
			s.closeRecv()
			break
		}
		v, err := s.recv()
		if err != nil {
			assert.ErrorIs(t, err, io.EOF)
			break
		}
		t.Log(v)
	}

	wg.Wait()
}

func TestStreamCopy(t *testing.T) {
	s := newStream[string](10)
	srs := s.asReader().Copy(2)

	s.send("a", nil)
	s.send("b", nil)
	s.send("c", nil)
	s.closeSend()

	defer func() {
		for _, sr := range srs {
			sr.Close()
		}
	}()

	for {
		v, err := srs[0].Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		t.Log("copy 01 recv", v)
	}

	for {
		v, err := srs[1].Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		t.Log("copy 02 recv", v)
	}

	for {
		v, err := s.recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		t.Log("recv origin", v)
	}

	t.Log("done")
}

func TestNewStreamCopy(t *testing.T) {
	t.Run("test one index recv channel blocked while other indexes could recv", func(t *testing.T) {
		s := newStream[string](1)
		scp := s.asReader().Copy(2)

		var t1, t2 time.Time

		go func() {
			s.send("a", nil)
			t1 = time.Now()
			time.Sleep(time.Millisecond * 200)
			s.send("a", nil)
			s.closeSend()
		}()
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			defer func() {
				scp[0].Close()
				wg.Done()
			}()

			for {
				str, err := scp[0].Recv()
				if err == io.EOF {
					break
				}

				assert.NoError(t, err)
				assert.Equal(t, str, "a")
			}
		}()

		go func() {
			defer func() {
				scp[1].Close()
				wg.Done()
			}()

			time.Sleep(time.Millisecond * 100)
			for {
				str, err := scp[1].Recv()
				if err == io.EOF {
					break
				}

				if t2.IsZero() {
					t2 = time.Now()
				}

				assert.NoError(t, err)
				assert.Equal(t, str, "a")
			}
		}()

		wg.Wait()

		assert.True(t, t2.Sub(t1) < time.Millisecond*200)
	})

	t.Run("test one index recv channel blocked and other index closed", func(t *testing.T) {
		s := newStream[string](1)
		scp := s.asReader().Copy(2)

		go func() {
			s.send("a", nil)
			time.Sleep(time.Millisecond * 200)
			s.send("a", nil)
			s.closeSend()
		}()

		wg := sync.WaitGroup{}
		wg.Add(2)

		//buf := scp[0].csr.parent.mem.buf
		go func() {
			defer func() {
				scp[0].Close()
				wg.Done()
			}()

			for {
				str, err := scp[0].Recv()
				if err == io.EOF {
					break
				}

				assert.NoError(t, err)
				assert.Equal(t, str, "a")
			}
		}()

		go func() {
			time.Sleep(time.Millisecond * 100)
			scp[1].Close()
			scp[1].Close() // try close multiple times
			wg.Done()
		}()

		wg.Wait()

		//assert.Equal(t, 0, buf.Len())
	})

	t.Run("test long time recv", func(t *testing.T) {
		s := newStream[int](2)
		n := 1000
		go func() {
			for i := 0; i < n; i++ {
				s.send(i, nil)
			}

			s.closeSend()
		}()

		m := 100
		wg := sync.WaitGroup{}
		wg.Add(m)
		copies := s.asReader().Copy(m)
		for i := 0; i < m; i++ {
			idx := i
			go func() {
				cp := copies[idx]
				l := 0
				defer func() {
					assert.Equal(t, 1000, l)
					cp.Close()
					wg.Done()
				}()

				for {
					exp, err := cp.Recv()
					if err == io.EOF {
						break
					}

					assert.NoError(t, err)
					assert.Equal(t, exp, l)
					l++
				}
			}()
		}

		wg.Wait()
		//memo := copies[0].csr.parent.mem
		//assert.Equal(t, true, memo.hasFinished)
		//assert.Equal(t, 0, memo.buf.Len())
	})

	t.Run("test closes", func(t *testing.T) {
		s := newStream[int](20)
		n := 1000
		go func() {
			for i := 0; i < n; i++ {
				s.send(i, nil)
			}

			s.closeSend()
		}()

		m := 100
		wg := sync.WaitGroup{}
		wg.Add(m)

		wgEven := sync.WaitGroup{}
		wgEven.Add(m / 2)

		copies := s.asReader().Copy(m)
		for i := 0; i < m; i++ {
			idx := i
			go func() {
				cp := copies[idx]
				l := 0
				defer func() {
					cp.Close()
					wg.Done()
					if idx%2 == 0 {
						wgEven.Done()
					}
				}()

				for {
					if idx%2 == 0 && l == idx {
						break
					}

					exp, err := cp.Recv()
					if err == io.EOF {
						break
					}

					assert.NoError(t, err)
					assert.Equal(t, exp, l)
					l++
				}
			}()
		}

		wgEven.Wait()
		wg.Wait()
		assert.Equal(t, m, int(copies[0].csr.parent.closedNum))
	})

	t.Run("test reader do no close", func(t *testing.T) {
		s := newStream[int](20)
		n := 1000
		go func() {
			for i := 0; i < n; i++ {
				s.send(i, nil)
			}

			s.closeSend()
		}()

		m := 4
		wg := sync.WaitGroup{}
		wg.Add(m)

		copies := s.asReader().Copy(m)
		for i := 0; i < m; i++ {
			idx := i
			go func() {
				cp := copies[idx]
				l := 0
				defer func() {
					wg.Done()
				}()

				for {
					exp, err := cp.Recv()
					if err == io.EOF {
						break
					}

					assert.NoError(t, err)
					assert.Equal(t, exp, l)
					l++
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, 0, int(copies[0].csr.parent.closedNum)) // not closed
	})

}

func checkStream(s *StreamReader[int]) error {
	defer s.Close()

	for i := 0; i < 10; i++ {
		chunk, err := s.Recv()
		if err != nil {
			return err
		}
		if chunk != i {
			return fmt.Errorf("receive err, expected:%d, actual: %d", i, chunk)
		}
	}
	_, err := s.Recv()
	if err != io.EOF {
		return fmt.Errorf("close chan fail")
	}
	return nil
}

func testStreamN(cap, n int) error {
	s := newStream[int](cap)
	go func() {
		for i := 0; i < 10; i++ {
			s.send(i, nil)
		}
		s.closeSend()
	}()

	vs := s.asReader().Copy(n)
	err := checkStream(vs[0])
	if err != nil {
		return err
	}

	vs = vs[1].Copy(n)
	err = checkStream(vs[0])
	if err != nil {
		return err
	}
	vs = vs[1].Copy(n)
	err = checkStream(vs[0])
	if err != nil {
		return err
	}
	return nil
}

func TestCopy(t *testing.T) {
	for i := 0; i < 10; i++ {
		for j := 2; j < 10; j++ {
			err := testStreamN(i, j)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCopy5(t *testing.T) {
	s := newStream[int](0)
	go func() {
		for i := 0; i < 10; i++ {
			closed := s.send(i, nil)
			if closed {
				fmt.Printf("has closed")
			}
		}
		s.closeSend()
	}()
	vs := s.asReader().Copy(5)
	time.Sleep(time.Second)
	defer func() {
		for _, v := range vs {
			v.Close()
		}
	}()
	for i := 0; i < 10; i++ {
		chunk, err := vs[0].Recv()
		if err != nil {
			t.Fatal(err)
		}
		if chunk != i {
			t.Fatalf("receive err, expected:%d, actual: %d", i, chunk)
		}
	}
	_, err := vs[0].Recv()
	if err != io.EOF {
		t.Fatalf("copied stream reader cannot return EOF")
	}
	_, err = vs[0].Recv()
	if err != io.EOF {
		t.Fatalf("copied stream reader cannot return EOF repeatedly")
	}
}

func TestStreamReaderWithConvert(t *testing.T) {
	s := newStream[int](2)

	var cntA int
	var e error

	convA := func(src int) (int, error) {
		if src == 1 {
			return 0, fmt.Errorf("mock err")
		}

		return src, nil
	}

	sta := StreamReaderWithConvert[int, int](s.asReader(), convA)

	s.send(1, nil)
	s.send(2, nil)
	s.closeSend()

	defer sta.Close()

	for {
		item, err := sta.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			e = err
			continue
		}

		cntA += item
	}

	assert.NotNil(t, e)
	assert.Equal(t, cntA, 2)
}

func TestArrayStreamCombined(t *testing.T) {
	asr := &StreamReader[int]{
		typ: readerTypeArray,
		ar: &arrayReader[int]{
			arr:   []int{0, 1, 2},
			index: 0,
		},
	}

	s := newStream[int](3)
	for i := 3; i < 6; i++ {
		s.send(i, nil)
	}
	s.closeSend()

	nSR := MergeStreamReaders([]*StreamReader[int]{asr, s.asReader()})

	record := make([]bool, 6)
	for i := 0; i < 6; i++ {
		chunk, err := nSR.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if record[chunk] {
			t.Fatal("record duplicated")
		}
		record[chunk] = true
	}

	_, err := nSR.Recv()
	if err != io.EOF {
		t.Fatal("reader haven't finish correctly")
	}

	for i := range record {
		if !record[i] {
			t.Fatal("record missing")
		}
	}
}

func TestMultiStream(t *testing.T) {
	var sts []*stream[int]
	sum := 0
	for i := 0; i < 10; i++ {
		size := rand.Intn(10) + 1
		sum += size
		st := newStream[int](size)
		for j := 1; j <= size; j++ {
			st.send(j&0xffff+i<<16, nil)
		}
		st.closeSend()
		sts = append(sts, st)
	}
	mst := newMultiStreamReader(sts)
	receiveList := make([]int, 10)
	for i := 0; i < sum; i++ {
		chunk, err := mst.recv()
		if err != nil {
			t.Fatal(err)
		}
		if receiveList[chunk>>16] >= chunk&0xffff {
			t.Fatal("out of order")
		}
		receiveList[chunk>>16] = chunk & 0xffff
	}
	_, err := mst.recv()
	if err != io.EOF {
		t.Fatal("end stream haven't return EOF")
	}
}

// 微型例子：两个子 reader 读取同一上游的第一个元素，应当拿到完全相同的值，且后续序列一致。
func TestCopyFirstElementSharedBetweenTwoChildren(t *testing.T) {
	s := newStream[int](0)
	copies := s.asReader().Copy(2)

	defer func() {
		for _, cp := range copies {
			cp.Close()
		}
	}()

	// 上游依次产生 1、2
	go func() {
		s.send(1, nil)
		time.Sleep(20 * time.Millisecond)
		s.send(2, nil)
		s.closeSend()
	}()

	// 子0先读到第一个元素
	v0a, err := copies[0].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v0a)

	// 稍后子1再来读第一个元素，也应拿到相同的值
	time.Sleep(5 * time.Millisecond)
	v1a, err := copies[1].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v1a)

	// 继续各自读取下一个位置，应同为 2
	v0b, err := copies[0].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 2, v0b)

	v1b, err := copies[1].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 2, v1b)

	// 均到 EOF
	_, err = copies[0].Recv()
	assert.ErrorIs(t, err, io.EOF)
	_, err = copies[1].Recv()
	assert.ErrorIs(t, err, io.EOF)
}

// 变体1：由子1先触发第一次填充，再由子0读取同一位置
func TestCopyFirstElementChild1First(t *testing.T) {
	s := newStream[int](0)
	copies := s.asReader().Copy(2)

	defer func() {
		for _, cp := range copies {
			cp.Close()
		}
	}()

	go func() {
		s.send(1, nil)
		time.Sleep(10 * time.Millisecond)
		s.send(2, nil)
		s.closeSend()
	}()

	// 子1先读到第一个位置
	v1a, err := copies[1].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v1a)

	// 子0随后读同一位置，应也为 1
	v0a, err := copies[0].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v0a)

	// 两者继续读取下一位置，均为 2
	v1b, err := copies[1].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 2, v1b)

	v0b, err := copies[0].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 2, v0b)

	// EOF
	_, err = copies[0].Recv()
	assert.ErrorIs(t, err, io.EOF)
	_, err = copies[1].Recv()
	assert.ErrorIs(t, err, io.EOF)
}

// 变体2：子0读取首元素后立即关闭，子1继续读到 EOF，且关闭计数正确
func TestCopyOneChildCloseEarlyOtherContinues(t *testing.T) {
	s := newStream[int](0)
	copies := s.asReader().Copy(2)

	// 便于断言 closedNum，这里拿到 parent 引用
	parent := copies[0].csr.parent

	go func() {
		s.send(1, nil)
		s.send(2, nil)
		s.send(3, nil)
		s.closeSend()
	}()

	// 子0读第一个元素后立刻关闭
	v0a, err := copies[0].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v0a)
	copies[0].Close()

	// 提前关闭不影响子1继续读取完整序列
	v1a, err := copies[1].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 1, v1a)

	v1b, err := copies[1].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 2, v1b)

	v1c, err := copies[1].Recv()
	assert.NoError(t, err)
	assert.Equal(t, 3, v1c)

	_, err = copies[1].Recv()
	assert.ErrorIs(t, err, io.EOF)

	// 此时仅一个子关闭
	assert.Equal(t, 1, int(parent.closedNum))

	// 关闭另一个子后，closedNum==2
	copies[1].Close()
	assert.Equal(t, 2, int(parent.closedNum))
}

// 压力测试：两个子 reader 并发读取大量元素，验证顺序一致且无死锁
func TestCopyTwoChildrenHighConcurrency(t *testing.T) {
	s := newStream[int](8)
	N := 2000

	// 生产者：随机节奏发送 0..N-1
	go func() {
		for i := 0; i < N; i++ {
			// 随机短暂睡眠，制造交错
			if i%7 == 0 {
				time.Sleep(time.Microsecond * time.Duration(rand.Intn(200)))
			}
			s.send(i, nil)
		}
		s.closeSend()
	}()

	copies := s.asReader().Copy(2)
	defer func() {
		for _, cp := range copies {
			cp.Close()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	consume := func(cp *StreamReader[int]) {
		defer wg.Done()
		expect := 0
		for {
			// 随机短暂睡眠，模拟不同步的消费节奏
			if expect%5 == 0 {
				time.Sleep(time.Microsecond * time.Duration(rand.Intn(200)))
			}
			v, err := cp.Recv()
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			assert.Equal(t, expect, v)
			expect++
		}
		assert.Equal(t, N, expect)
	}

	go consume(copies[0])
	go consume(copies[1])

	wg.Wait()
}
