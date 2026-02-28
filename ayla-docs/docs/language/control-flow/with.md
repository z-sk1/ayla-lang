# With Statement
the `with` statement evaluates an expression and makes the result available inside a block as the special name `it`.

`it` only exists inside the with statement block

```ayla
with x {
    putln(it)
}
```
inside the block, `it` refers to the value of x.


## it Is Read-Only
inside a with block, `it` behaves like a constant

```ayla
with x {
    it = 10   // not allowed
}
```
you cannot reassign it

## nested with statements
Nested `with` blocks create a new `it`, which shadows the outer one.

```ayla
with a {
    with b {
        putln(it) // refers to b
    }
}
```
the inner `it` hides the outer `it` for the duration of the inner block

## special uses
since `with` allows any expression, you can do some odd stuff like this

```ayla
with sum(1, 3) {
    putln(it)
}
```
> output:
```
4
```

instead of

```ayla
x := sum(1, 3)
putln(x)
```
> output:
```
4
```