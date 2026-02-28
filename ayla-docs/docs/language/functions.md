# Functions
to declare a function use `fun`

declare them and call them like this

```ayla
fun hi() {
    putln("hi")
}

hi()
```
> output: hi

ayla requires static typing so you need to specify parameter types

## return
return has been renamed to `back`

```ayla
fun add(x int, y int) (int) {
    back x + y
}

put(add(5, 7))
```
output: 12

## multiple return values
ayla also supports multiple return values
```ayla
fun operation(x int, y int) (int, int) {
    back x + y, x - y
}

putln(operation(4, 5))
```
> output: 9 -1
