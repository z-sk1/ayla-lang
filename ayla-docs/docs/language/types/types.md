# Types

# built in types
Currently ayla provides these built in types

- `int`
- `float`
- `string`
- `bool`
- `error`
- `thing`
- `nil`

## type annotation
you can specify a variable's type during declaration

```ayla
egg x int = 3
putln(x)
```

type annotations work for both `egg` (mutable) and `rock` (constant)

```ayla
rock name string = "ayla"
```

## type inferrence
if no type is specified, ayla infers it from the assigned value

```ayla
egg x = 3 // inferred as int
egg y = 3.0 // inferred as float
egg z = "hello" // inferred as string
```

## default values

when a variable is declared without an initial value, it receives one based on its type

```ayla
egg a int // 0
egg b float // 0.0
egg c string // ""
egg d bool // no
egg e error // nil
egg f // nil
```

| Type     | Default |
| -------- | ------- |
| `int`    | `0`     |
| `float`  | `0.0`   |
| `string` | `""`    |
| `bool`   | `no`    |
| `error`  | `nil`   |
| untyped  | `nil`   |


for example
```ayla
egg x int
putln(x)
```
> output: 0

## type mismatches
if you assign a value of the wrong type it results in a runtime error

```ayla
egg x string = 5
```
> output: runtime error at 1:5: type mismatch: 'int' assigned to a 'string'

## multiple type annotation 
when declaring multiple variables, one annotation at the end applies to the rest of the variables

```ayla
egg a, b, c int

putln(typeof(a), typeof(b), typeof(c))
```
> output: int int int

## type casting

use function style syntax to convert values between types

```ayla
egg x float = 5.3

egg y int = int(x)

putln(y)
```
> output: 5

here we convert a float to an int


## invalid casts
however notice that some casts cant be performed!

for example, you cant cast a `string` to an `int` since `strings` are not numerical values
```ayla
egg x string = "hi"

egg y int = int(x)
```
> output: runtime error at 3:16: int type cast does not support 'string'

## parsing vs casting

type casting converts between compatible numeric types (such as float to int).

to convert non-numeric types (like string) into numbers, you must use a parsing function:
```ayla
egg x string = "2"
egg y int = toInt(x)
putln(y)
```
> output: 2

parsing interprets the contents of the string and converts it into a numeric value.

## thing type
the `thing` type is equivalent to `any` from TypeScript or Go

you can assign any value to it

```ayla
egg x thing = 2

put(x)
```
> output: 2

## type assertion

but, you must use `type assertion` to do operations with `thing`
```ayla
egg x thing = 2

put(x.(int) + 1)
```
> output: 3

otherwise you will come across a `Runtime error`
```ayla
egg x thing = 2

put(x + 1)
```
> output: runtime error at 3:11: cannot use 'thing' in operations, assert a type first

type assertions protect against type mismatches, so they produce a `Runtime error` when a `thing` is asserted incorrectly
```ayla
egg x thing = 2

put(x.(string) + "2")
```
> output: runtime error at 3:14: type mismatch: 'int' asserted as 'string'
