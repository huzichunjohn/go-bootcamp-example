package main

import (
    "fmt"
    "math/rand"
    "time"
    "sync"
)

func main() {
    repeatFn := func(done <-chan interface{}, fn func() interface{}) <-chan interface{} {
	valueStream := make(chan interface{})
	go func() {
	    defer close(valueStream)
	    for {
		select {
		case <-done:
		    return
		case valueStream <- fn():
		}
	    }
	}()
	return valueStream
    }

    take := func(done <-chan interface{}, valueStream <-chan int, num int) <-chan interface{} {
	takeStream := make(chan interface{})
	go func() {
	    defer close(takeStream)
	    for i := 0; i < num; i++ {
		select {
		case <-done:
		    return
		case takeStream <- <- valueStream:
		}
	    }
	}()
	return takeStream
    }

    toInt := func(done <-chan interface{}, valueStream <-chan interface{}) <-chan int {
	intStream := make(chan int)
	go func() {
	    defer close(intStream)
	    for v := range valueStream {
		select {
		case <-done:
		     return
		case intStream <- v.(int):
		}
	    }
	}()
	return intStream
    }

    primeFinder := func(done <-chan interface{}, valueStream <-chan int) <-chan int {
	intStream := make(chan int)
	go func() {
	    defer close(intStream)
	    for v := range valueStream {
		select {
		case <-done:
		    return
		default:
		    isPrime := true
		    for i := 2; i < v; i++ {
			if v%i == 0 {
			    isPrime = false
			    break
			}
		    }
		    if isPrime {
			intStream <- v	
		    }
		}
	    }
	}()
	return intStream
    }

    fanIn := func(done <-chan interface{}, channels ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	multiplexedStream := make(chan int)

	multiplex := func(c <-chan int) {
	    defer wg.Done()
	    for i := range c {
		select {
		case <-done:
		    return
		case multiplexedStream <- i:
		}
	    }
	}

	wg.Add(len(channels))
	for _, c := range channels {
	    go multiplex(c)
	}
	
	go func() {
	    wg.Wait()
	    close(multiplexedStream)
	}()	

	return multiplexedStream
    }

    done := make(chan interface{})
    defer close(done)

    start := time.Now()

    rand := func() interface{} { return rand.Intn(500000) }

    randIntStream := toInt(done, repeatFn(done, rand))

    numFinders := 16
    fmt.Printf("Spinning up %d prime finders.\n", numFinders)
    finders := make([]<-chan int, numFinders)
    fmt.Println("Primes:")
    for i := 0; i < numFinders; i++ {
	finders[i] = primeFinder(done, randIntStream)
    }

    for prime := range take(done, fanIn(done, finders...), 10) {
	fmt.Printf("\t%d\n", prime)
    }
    
    fmt.Printf("Search took: %v\n", time.Since(start))
}
