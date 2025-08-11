# King Conc

A Go library with thread-safe concurrency tools I made primarily for my job, as I found myself rewriting or copy-pasting the same utilities from one project to another.

If it can help someone else, then that’s a plus!

Almost all code was written by hand, but most tests were generated using [OpenCode](https://github.com/sst/opencode).

## Maintainer

[@BigTony299](https://github.com/BigTony299)

## Contributing

Contributions, issues, and feature requests are welcome!
Feel free to submit a pull request.

## Features

### Worker Pools

Worker pools control how many goroutines run at once. Instead of creating unlimited goroutines, worker pools limit the number and reuse them.

- **Static pools** - Fixed number of workers
- **Dynamic pools** - Creates workers as needed  
- **Sync pools** - Runs tasks immediately in your current goroutine

Use worker pools when you have many tasks but want to limit memory usage.

```go
// this will ready 4 workers at initialization
pool := NewWorkerPoolStatic(4) 

// this will create workers on demand
pool := NewWorkerPoolDynamic()

// this will execute tasks synchronously in the current goroutine
pool := NewWorkerPoolSync()

pool.Go(func() { fmt.Println("Task 1") })
pool.Go(func() { fmt.Println("Task 2") })
```

### Broadcasting

Broadcasting sends the same message to multiple listeners at once. Each listener gets their own channel to receive messages.

- **Buffered broadcasting** - Messages queue up if listeners are slow
- **Unbuffered broadcasting** - Messages are dropped if listeners aren't ready
- **Concurrent delivery** - Uses worker pools to send messages

Use broadcasting when you need to notify multiple parts of your application about events.

```go
// this broadcaster uses default settings (unbuffered, sync pool)
broadcaster := NewBroadcaster[string](BroadcasterOptions{})

// this broadcaster uses a custom worker pool for async delivery
broadcaster := NewBroadcaster[string](BroadcasterOptions{
    WorkerPool: Some[WorkerPool](NewWorkerPoolStatic(4)),
})

// this broadcaster has buffered channels to prevent message dropping
broadcaster := NewBroadcaster[string](BroadcasterOptions{
    BufferSize: Some(10),
})

// this broadcaster uses both custom pool and buffering
broadcaster := NewBroadcaster[string](BroadcasterOptions{
    WorkerPool: Some[WorkerPool](NewWorkerPoolDynamic()),
    BufferSize: Some(5),
})

ch := broadcaster.Subscribe()

broadcaster.Broadcast("Hello everyone!")

msg := <-ch // "Hello everyone!"

// cleanup when done
broadcaster.Unsubscribe(ch)
broadcaster.Close()
```

### Atomic Values

Atomic values let multiple goroutines safely read and change the same data. Multiple goroutines can access the data without race conditions.

- **Thread-safe operations** - Multiple goroutines can access safely
- **Optional values** - Can be empty or contain data
- **Atomic numbers** - Support for math operations like increment/decrement
- **Callbacks** - Run functions when values reach certain conditions

Use atomic values when you need to share data between goroutines.

```go
// this creates an atomic container with an initial value
atomic := NewAtomic("initial")

// this creates an atomic number with math operations
counter := NewAtomicNumber(0)

// basic number operations
counter.Increment() // 1
counter.Add(5)      // 6
counter.Decrement() // 5
counter.Sub(2)      // 3

// comparison operations
counter.Eq(3)  // true
counter.Gt(2)  // true  
counter.Lte(5) // true

// atomic container operations
atomic.Put("new value")

// update with a function
atomic.Update(func(current string) string {
    return current + " updated"
})

// compare with a predicate
hasLength := atomic.Compare(func(s string) bool {
    return len(s) > 10
})

// this "takes" the value from the container, marking it empty
value := atomic.Take() // Some("new value updated")

// callbacks execute when conditions are met
counter.Once(func(n int) bool { return n >= 10 }, func() {
    fmt.Println("Reached 10!")
})

counter.Every(func(n int) bool { return n%5 == 0 }, func() {
    fmt.Println("Multiple of 5!")
})

counter.Add(7) // triggers "Reached 10!" and "Multiple of 5!"
```

### Queues

Queues pass work between goroutines in order. The first item added is the first one removed.

- **Thread-safe operations** - Multiple goroutines can add/remove items safely
- **Blocking operations** - Wait for new items
- **Non-blocking operations** - Return immediately if queue is empty

Use queues when you need to process work in order.

```go
// this creates a thread-safe queue
queue := NewQueue[int]()

queue.Push(1)
queue.Push(2)
queue.Push(3)

// check how many items are in queue
count := queue.Count() // 3

// look at next item without removing it
next := queue.Peek() // Some(1)

// remove and return next item
item := queue.Pop() // Some(1)

// queue now has [2, 3]
fmt.Println(queue.Count()) // 2

// this blocks until an item is available
// use in a separate goroutine for non-blocking behavior
next := queue.Next() // 2

// queue is now [3]
```

### Group Pools

Group pools run many tasks and collect all their results or errors together.

- **Error collection** - Gather all errors from failed tasks
- **Completion waiting** - Wait for all tasks to finish
- **Result aggregation** - Collect outputs from all tasks

Use group pools when you need to run multiple tasks and want to know when they're all done.

```go
// this uses a static worker pool with 4 workers
pool := NewGroupPool(GroupPoolConfig{Limit: Some(4)})

// this uses a dynamic worker pool
pool := NewGroupPool(GroupPoolConfig{Limit: None[int]()})

errors := pool.Go(
    func() error { return processFile("file1.txt") },
    func() error { return processFile("file2.txt") },
)

// collect all errors from the channel
for err := range errors {
    if err != nil {
        fmt.Println("Error:", err)
    }
}
```

### Optional Types

Optional types handle values that might not exist. Instead of guessing if something has a value, you check explicitly.

- **Safe value handling** - No nil pointer panics
- **Clear intent** - Makes it obvious when values might be missing
- **Chainable operations** - Work with optional values

Use optional types when dealing with data that might be missing.

```go
// this creates an optional with a value
value := Some(42)

// this creates an empty optional
empty := None[int]()

// check if value exists
if value.Some() {
    fmt.Println("Has value:", value.Raw()) // 42
}

// check if value is missing
if empty.None() {
    fmt.Println("No value present")
}

// get value with boolean indicator
val, ok := value.Get()
if ok {
    fmt.Println("Value:", val) // 42
}

// get value or use default
result := empty.GetOrDefault(100) // 100

// work with different types
name := Some("John")
missing := None[string]()

fmt.Println(name.GetOrDefault("Unknown"))    // "John"
fmt.Println(missing.GetOrDefault("Unknown")) // "Unknown"

// work with structs
type User struct {
    ID   int
    Name string
}

user := Some(User{ID: 1, Name: "Alice"})
if user.Some() {
    u := user.Raw()
    fmt.Printf("User: %s (ID: %d)\n", u.Name, u.ID)
}
```
