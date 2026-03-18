# Arrays

An array is a fixed-size collection of values of the same type.

In Ayla, arrays have a length that is part of their type.

## declaring an array
the syntax for an array type is:
```ayla
[length]Type
```
Example:
```ayla
egg x [5]int
```
This declares an array of:
```ayla
Length: 5

Type: int
```
Since arrays have zero values, this is equivalent to:

```ayla
egg x [5]int = [5]int{}
```
the default value for each element is the `zero value` of its `type`.

for `int`, that is 0

## array literals
You can initialize an array using a literal:
```ayla
egg x = [5]int{1, 2, 3, 4, 5}
putln(x)
```
each value inside `{}` fills the `array` in order.

## accessing elements
Arrays use zero-based indexing

```ayla
egg x = [5]int{10, 20, 30, 40, 50}

putln(x[0]) // 10
putln(x[2]) // 30
```
> output:
```
10
30
```

the first element is at index 0

## modifying elements

```ayla
egg x = [5]int{1, 2, 3, 4, 5}
x[0] = 100

putln(x[0])
```

## array length is part of the type
this is very important:

```ayla
egg a [5]int
egg b [3]int
```

these are *different* types.

you cannot assign one to the other

```ayla
a = b   // type mismatch
```

Because `[5]int` and `[3]int` are not the same type

## zero value
The zero value of an array is an array where every element is the zero value of its type

Example:

```ayla
egg x [3]string
putln(x)
```

This is equivalent to:

```ayla
[3]string{"", "", ""}
```
