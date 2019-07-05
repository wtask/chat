package background

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

func randInt() int {
	sign := rand.Intn(100)
	value := rand.Intn(math.MaxInt32)
	if sign < 50 {
		return -value
	}
	return value
}

func producer(s *Scope, id string, data chan<- int) {
	defer s.Done()
	for {
		select {
		case data <- randInt():
		case <-s.Context().Done():
			fmt.Println(id, "done")
			return
		}
	}
}

func consumer(s *Scope, id string, data <-chan int) {
	defer s.Done()
	for {
		select {
		case _, ok := <-data:
			if !ok {
				fmt.Println(id, "exited on closed data channel")
				return
			}
		case <-s.Context().Done():
			fmt.Println(id, "done")
			return
		}
	}
}

func ExampleScope() {
	data1, data2, data3 := make(chan int), make(chan int), make(chan int)

	write1, cancelWrite1 := NewScope()
	read1, cancelRead1 := NewScope()
	write2, cancelWrite2 := NewScope()
	read3, cancelRead3 := NewScope()

	write1.Add(1)
	go producer(write1, "DATA-1 *PRODUCER*", data1)
	read1.Add(1)
	go consumer(read1, "DATA-1 *CONSUMER*", data1)

	write2.Add(1)
	go producer(write2, "DATA-2 *PRODUCER*", data2) // blocked due to no consumer for data2

	read3.Add(1)
	go consumer(read3, "DATA-3 *CONSUMER*", data3) // blocked due to no producer for data3

	time.Sleep(50 * time.Millisecond)

	// Cancel all background scopes in desired order:
	cancelWrite2()
	cancelRead3()
	cancelWrite1()
	cancelRead1()

	// Output:
	//
	// DATA-2 *PRODUCER* done
	// DATA-3 *CONSUMER* done
	// DATA-1 *PRODUCER* done
	// DATA-1 *CONSUMER* done
}

func ExampleScope_severalMembers() {
	data := make(chan int)

	scope, cancel := NewScope()

	scope.Add(3)
	go producer(scope, "*PRODUCER-1*", data)
	go producer(scope, "*PRODUCER-2*", data)
	go producer(scope, "*PRODUCER-3*", data)

	time.Sleep(50 * time.Millisecond)

	cancel()

	// Unordered output:
	//
	// *PRODUCER-1* done
	// *PRODUCER-2* done
	// *PRODUCER-3* done
}

func ExampleScope_expiredOrActive() {
	scope1, cancel1 := NewScope()
	defer cancel1()
	scope2, cancel2 := NewScope()
	cancel2()
	// expired condition is: err != nil, if false than scope is in active state
	fmt.Println(scope1.Context().Err() != nil, scope2.Context().Err() != nil)
	// active condition is: err == nil, if false than scope is in expired state
	fmt.Println(scope1.Context().Err() == nil, scope2.Context().Err() == nil)

	// Output:
	// false true
	// true false
}
