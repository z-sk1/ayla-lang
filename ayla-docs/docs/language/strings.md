# Strings

## string concatenation
you can concatenate (combine) strings using the `+` operator.

```ayla
say a = "hello "
say b = "world"

put(a + b)
```
> output: hello world

you can also concatenate strings with other types by parsing.
```ayla 
put(toString(4) + toString(2))
```
> output: 42

## string interpolation
you can also interpolate strings using `${}`

> unlike JavaScript, you just use the normal quotation marks, "", not ``

```ayla
say rand = randi(10)

put("Random number: ${rand}")
```
> output: 0 - 10

## string indexing
you can index into strings like arrays and slices

```ayla 
say text = "Hello"

putln(text[0])
```
> output: H
