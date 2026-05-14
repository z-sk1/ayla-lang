# Start Statement

The start statement runs a function concurrently in a new lightweight thread of execution.

## syntax

this the traditional syntax
```ayla
start function()
```

but you can also use a block format like this
```ayla
start {}
```

note that this is just syntactical sugar for:
```ayla
start fun(){}()
```

## example

```ayla
fun work() {
    putln("working...")
}

start work()
putln("main continues")
```

## behavior
- start does not block
- execution continues immediately after spawning
- the started function runs independently

## Notes
- similar to go in Go
- no return value is captured directly
- use channels to communicate results