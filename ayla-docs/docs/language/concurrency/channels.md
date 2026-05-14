# Channels
Channels are used to communicate between concurrent routines safely.

## creating a Channel
```ayla
say ch = make(chan int)
```

this will create a channel that can send/receive int values.

## sending Values
```ayla
ch <- 5
```
Sends 5 into the channel.

## receiving Values
```ayla
x := <-ch
```
Receives a value from the channel.

## Receiving with Status (ok)
```ayla
x, ok := <-ch
```

- x will be the received value
- ok wlll be a boolean, yes if successful, no if channel is closed

## Example
```ayla
ch := make(chan int)

start fun() {
    ch <- 42
}()

val := <-ch
putln(val)
```
## communication Example
```ayla
say ch = make(chan int)

start fun() {
    ch <- 10
}()

x := <-ch
putln(x)
```
## important rules

- Receiving from an empty channel blocks
- Sending to a channel with no receiver blocks
- Use start to avoid deadlocks
- Channels are the primary way to share data between concurrent taskss

