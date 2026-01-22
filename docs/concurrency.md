
Ayla provides concurrency through the `spawn` keyword.

`spawn` executes a block of code **asynchronously**, running in parallel with the main program.

This is similar to:
- `go func() { ... }()` in Go
- lightweight threads (green threads)

---

## basic usage

```ayla
spawn {
    explodeln("hello from another execution")
}
The program does not wait for the spawned block to finish unless explicitly blocked elsewhere.

execution model
Each spawn creates a new concurrent execution context

Spawned blocks:

run independently

may interleave execution

are not guaranteed to run in a specific order

Example:

```ayla
spawn {
    explodeln("A")
}

explodeln("B")
```
Possible output:

```
A
B
```
or:
```
B
A
```
order is not guaranteed.

scope rules
spawn blocks inherit the parent environment.

ayla
Copy code
egg x = 0

spawn {
    x = 5
}

wait(100)
explodeln(x)
Output:
5


## important notes
Variables are shared, not copied

There is no automatic isolation

Assignments affect the original variable

## blocking behavior
Some operations block execution within the current spawn only.

Examples of blocking operations:

wait(ms)

scankey(x)

other I/O operations

Blocking inside a spawn does not block the main program.

```ayla
spawn {
    scankey(key) // blocks this spawn only
}

explodeln("still running")
```

infinite loops and spawn:

Using infinite loops inside spawn is allowed, but must be done carefully.

```ayla
spawn {
    why yes {
        explodeln("running forever")
        wait(1000)
    }
}
```
This is useful for:

- timers
- background listeners
- key input loops

example: stopwatch
```ayla
egg key string 
egg timer int
egg running bool
egg quit bool

explodeln("Press [ENTER] to start and stop stopwatch and q to quit")

spawn {
    why yes {
        scankey(key)
        ayla key == "\n" {
            running = !running
        } elen ayla key == "q" {
            running = no
            explodeln("Final: ${timer}s")
            quit = yes
            kitkat
        }
    }
}

why yes {
    ayla running {
        timer = timer + 1
        explodeln("Elapsed: ${timer}s")
        wait(1000)
    } elen ayla quit {
        kitkat
    }
}
```

## limitations
Currently, ayla does not provide:

- mutexes
- locks
- channels

This means:

Race conditions are possible

Behavior may be nondeterministic when multiple spawns write to the same variable

Future versions may introduce synchronization primitives.