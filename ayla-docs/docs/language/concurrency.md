# Concurrency

Ayla provides concurrency through the `spawn` keyword.

`spawn` executes a block of code **asynchronously**, running in parallel with the main program.

---

## basic usage

```ayla
spawn {
    putln("hello from another execution")
}
```
The program does not wait for the spawned block to finish unless explicitly blocked elsewhere.

## execution model
Each spawn creates a new concurrent execution context

Spawned blocks:
- run independently
- may interleave execution
- are not guaranteed to run in a specific order

Example:

```ayla
egg x = 0

spawn {
    x = x + 1
}

spawn {
    x = x + 1
}
```

So the final value of `x` is not guaranteed

## scope rules
spawn blocks inherit the `parent environment`.

```ayla
egg x = 0

spawn {
    x = 5
}

wait(100)
putln(x)
```
Output:
5


## important notes
- Variables are shared, not copied
- There is no automatic isolation
- Assignments affect the original variable

## blocking behavior
Some operations block execution within the current spawn only.

Examples of blocking operations:

- wait(ms)
- scankey(x)
- other I/O operations

Blocking inside a spawn does not block the main program

```ayla
spawn {
    scankey(key) // blocks this spawn only
}

putln("still running")
```

## infinite loops and spawn

Using infinite loops inside spawn is allowed, but must be done carefully.

```ayla
spawn {
    why yes {
        putln("running forever")
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

putln("Press [ENTER] to start and stop stopwatch and q to quit")

spawn {
    why yes {
        scankey(key)
        ayla key == "\n" {
            running = !running
        } elen ayla key == "q" {
            running = no
            putln("Final: ${timer}s")
            quit = yes
            kitkat
        }
    }
}

why yes {
    ayla running {
        timer = timer + 1
        putln("Elapsed: ${timer}s")
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
