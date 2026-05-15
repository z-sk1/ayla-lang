# Variables

A variable is a place for storing a value, a constant is a place for storing a value which can never be changed

## declaration and assignment
to declare a normal mutable, reassignable variable use `say`
```ayla
say x = "wowie"
```

variables can also be declared without an initial value, like so
```ayla
say x
```

to declare a constant, a variable which cannot be changed, use `keep`

```ayla
keep x = "i will never change"
```

if you try to assign a new value, you will across a `Runtime error`
```ayla
keep x = "i will never change"

x = "i want to change"
```
> output: runtime error at 3:2: cannot reassign to const: x

constants cannot be declared without an intitial value.
```ayla
keep x
```
> output: Runtime error at 1:5: const x must be initialised

## multi-declaration and multi-assignment
you can also assign and declare multiple variables at the same time

```ayla
keep a, b = 4, 2

putln("${a} ${b}")
```
> output: 4 2

you can also just declare them without an inital value like normal
```ayla
say a, b
```

and also like single declaration, using multi constant declaration you must initialise it
```ayla
keep a, b
```
> output: runtime error at 1:5: constants, a, b, must be initialised


same principles for multi assignment
```ayla
say a, b

a, b = 4, 2

putln("${a} ${b}")
```
> output: 4 2

## function return value destructuring
you can assign and declare multiple variables to function return values and it will destructure it
```ayla
fun operation(x int, y int) (int, int) {
    give x + y, x - y
}

say a, b int = operation(5, 3)

putln(a, b)
```
> output:
```
8
2
```
 
## declaration blocks
you can also do `declaration blocks` like in Go

```ayla
say (
  a = 1
  b = no
)

keep (
  x int = 4
  y bool = yes
)
```
