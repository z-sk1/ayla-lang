# Pointers and References
Ayla supports pointers and references, which allow you to work directly with the memory location of a value instead of the value itself.

Pointers are useful when you want a function to modify the original variable instead of a copy.

## pointer type syntax
```ayla
*Type
```
this declares a type of pointer to `Type`

so this:
```ayla
*int
```
would declare a pointer type to `int`

## creating a pointer
use `&` to get the `reference` (`address`) of a `variable`.

```ayla
egg x int = 10
egg p *int = &x

putln(x)
putln(p)
```
> output:
```
10
ptr(0x... -> 10)
```

### how it works
- `x` stores the value `10`
- `&x` returns the memory address of `x`
- `p` now holds a `pointer` to `x`

## dereferencing a pointer
to access the value stored at a `pointer`, use `*`.

```ayla
egg x int = 10
egg p *int = &x

putln(*p)
```
> output:
```
10
```

## what dereferencing does
- `p` stores the `address` of `x`
- `*p` reads the value stored at that `address`

So:
```ayla
*p == x
```

## modifying values through pointers
Pointers allow you to change the original variable.

```ayla
egg x int = 10
egg p *int = &x

*p = 20

putln(x)
```
> output:
```
20
```

## pointers in functions
Pointers are commonly used to allow functions to modify arguments.

```ayla
fun increment(n *int) {
    *n += 1
}

egg x int = 5
increment(&x)

putln(x)
```
> output:
```
6
```

## references vs values
normally, Ayla passes values by copy.

```ayla
fun change(x int) {
    x = 20
}

egg n int = 10
change(n)

putln(n)
```
> output:
```
10
```
this is because `x` is a copy of `n` in the new scope of the function, so changing `x` wont affect `n`

we can solve this by using a pointer
```ayla
fun change(x *int) {
    *x = 20
}

egg n int = 10
change(&n)

putln(n)
```
> output:
```
20
```
so now the function modifies the original `variable`

## important rules
- `&` gets the address of a variable
- `*` dereferences a pointer
- `*T` means pointer to type T

## example combining everything
```ayla
fun swap(a *int, b *int) {
    temp := *a
    *a = *b
    *b = temp
}

egg x int = 3
egg y int = 7

swap(&x, &y)

putln(x, y)
```
> output:
```
7 3
```
