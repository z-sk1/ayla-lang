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

## parameter types
ayla requires static typing so you need to specify parameter types

```ayla
fun hi(name string) {
    putln("hi " + name)
}

hi("Ziad")
```
> output:
```
hi Ziad
```

## return
return has been renamed to `back`

```ayla
fun add(x int, y int) (int) {
    back x + y
}

put(add(5, 7))
```
> output: 
```
12
```

## multiple return values
ayla also supports **multiple return values**
```ayla
fun operation(x int, y int) (int, int) {
    back x + y, x - y
}

putln(operation(4, 5))
```
> output: 
```
9 -1
```

## variadic parameters
Ayla supports **variadic parameters**, which allow a function to accept a variable number of arguments

Variadic parameters use `...` before the type

```ayla
fun sum(nums ...int) (int) {
    total := 0
    four _, n := range nums {
        total = total + n
    } 
    back total
}

putln(sum(1, 2, 3, 4))
```
> output:
```
10
```

### how it works
- `nums ...int` means:
  "zero or more int values"
- Inside the function, `nums` behaves like an `slice`
- You can `iterate` over it like a normal `slice`


You can also call it with no args:
```ayla
putln(sum())
```
> output:
```
0
```

## mixing normal and variadic parameters
A variadic parameter must be the last parameter in the function.

```ayla
fun greet(prefix string, names ...string) {
    four _, name := range names {
        putln(prefix + " " + name)
    }
}

greet("hi", "Ziad", "Ayla", "Elen")
```

> output:
```
hi Ziad
hi Ayla
hi Elen
```

## flattening (spreading arguments)
Sometimes you already have an `array` or a `slice` and want to pass its values into a variadic function.

For this, Ayla supports flattening using `...` when calling

```ayla
egg numbers = []int{1, 2, 3, 4}

putln(sum(numbers...))
```
output:
```
10
```
### what flattening does
- `numbers...` expands the array
- Each element becomes a separate argument

Without flattening:
```ayla
sum(numbers)   // type error
```
> output:
```

```

Because:

- `sum` expects `...int`
- `numbers` is a `slice`, not individual `ints`

### important rules
- A function can have only one `variadic parameter`
- The `variadic parameter` must be last
- Inside the function, it behaves like an `slice`
- `Flattening` (...) is only valid when calling `variadic functions`

## example combining everything
```ayla
fun printAll(values ...string) {
    four _, v := range values {
        putln(v)
    }
}

egg words = []string{"Ayla", "is", "cool"}

printAll("Hello")
printAll(words...)
```
> output:
```
Hello
Ayla
is
Cool
```