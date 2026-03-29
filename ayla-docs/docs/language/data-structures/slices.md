# Slices
A slice is a dynamic sequence of elements

Unlike arrays, slices:
- Do not have a fixed length in their type
- Can grow and shrink

## slice type
the syntax for a slice type is:
```ayla
[]Type
```
Example:
```ayla
egg x []int
```

this declares a slice of integers.

to declare a slice type use a type statement:
```ayla
type Ages []int
```

## slice literal
you can create a slice using a literal:
```ayla
x := []int{1, 2, 3}
putln(x)
```
unlike arrays, **you do not specify the length**.

## accessing elements
slices use zero-based indexing:
```ayla
x := []int{10, 20, 30}

putln(x[0])  // 10
putln(x[2])  // 30
```

## slicing an existing slice
You can create a new slice from an existing one:

```ayla
x := []int{1, 2, 3, 4, 5}

y := x[1:4]

putln(y)
```
> output:
```ayla
2
3
4
```

Notice how the first element of the sliced slice `y` is included, but the last one isnt

## zero value
the `zero value` of a slice is just an empty literal, since slices are `dynamic`

```ayla
egg x []int

putln(x)
```
> output:
```
[]
```
