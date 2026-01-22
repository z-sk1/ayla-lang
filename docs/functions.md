# functions

to declare a function use `fun`

return has been renamed to `back`, haha

```ayla
fun add(x, y) {
    back x + y
}

explode(add(5, 7))
```
output: 12

you **can** have a designated return type like this
```ayla
fun add(x, y) (int) {
    back x + y
}

explodeln(add(4, 2))
```
output: 6

you will encounter a `Runtime error` if you use the wrong type
```ayla
fun add(x, y) (string) {
    back x + y
}

explodeln(add(4, 1))
```
> output: runtime error at 5:14: return 1, expected string, got int

you can also add types to parameters
```ayla
fun add(x int, y int) (int) {
    back x + y
}
```

you will encounter a `Runtime error` as well if you use the wrong type for the parameter
```ayla
fun add(x int, y int) (int) {
    back x + y
}

egg sum = add("4", 2)
```
> output: runtime error at 5:14: paramteter 'x' expected int, got string