The select statement waits on multiple channel operations and executes the first one that becomes ready.

It is used to coordinate concurrent communication.

## Syntax
```ayla
select {
    when <channel_op> {
        // code
    }
    when <channel_op> {
        // code
    }
    otherwise {
        // optional default
    }
}
```
## Example
```ayla
say ch1 = chan int
say ch2 = chan int

start {
    ch1 <- 1
}

start {
    ch2 <- 2
}

select {
    when x := <-ch1 {
        putln("ch1:", x)
    }
    when y := <-ch2 {
        putln("ch2:", y)
    }
}
```
## Behavior
- waits until one of the channel operations is ready
- executes only one matching branch
- if multiple are ready → one is chosen randomly
- after executing a branch, the select exits

## blocking behavior

If there is no case `select` blocks, unless an otherwise case is provided

## otherwise (Default Case)
```ayla
select {
    when x := <-ch {
        putln(x)
    }
    otherwise {
        putln("nothing ready")
    }
}
```

## Receiving with ok
You can use the ok pattern:

```ayla
select {
    when x, ok := <-ch {
        ayla ok {
            putln("received:", x)
        } elen {
            putln("channel closed")
        }
    }
}
```
## Sending in select
You can also wait on send operations:

```ayla
select {
    when ch <- 10 {
        putln("sent value")
    }
}
```

## example: Timeout Pattern
```ayla
say ch = make(chan int)
say timeout = make(chan int)

start {
    timeout <- 1
}

select {
    when x := <-ch {
        putln("received:", x)
    }
    when <-timeout {
        putln("timeout")
    }
}
```