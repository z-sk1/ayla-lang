# Defer Statement

The `defer` statement schedules a function call to run when the surrounding function returns

## syntax

this is the traditional syntax
```ayla
defer function_call()
```

but you can also use a block format like this
```ayla
defer {}
```

note that this is just syntactical sugar for:
```ayla
defer fun(){}()
```

## example
```ayla
fun main() {
    defer putln("done")
    putln("working...")
}
```
> output:
```
working...
done
```

## behavior
- deferred calls run after the function finishes 
- execution order is last-in, first-out (LIFO)

## multiple defer
```ayla
fun main() {
    defer putln("first")
    defer putln("second")
    defer putln("third")
}
```
> output:
```
third
second
first
```
