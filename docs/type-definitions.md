# type definitions

## named types
named types are a completely new type, that have another type underneath.

here, `Age` has an underlying type of `int`
```ayla
type Age int
```

you can assign an int to it

```ayla
type Age int

egg a Age = 3
```

```ayla 
type Age int

egg a Age = 3

put(a)
```
> output: 3

you can also do it without type annotation by using a type cast

```ayla
type Age int

egg a = Age(3)

put(a)
```
> output: 3

the available types are:
- `int`
- `float`
- `string`
- `bool`
- `arr`
- `struct` - learn more about in [docs/structs.md](docs/structs.md)

## aliases
aliases are just a new name for a primitive type and are completely equivalent to them

```ayla
type Number = int
```

so you dont need to type cast to `Number` here since it is completely equivalent to `int`

```ayla
type Number = int

egg x Number = 5

putln(x)
```
> output: 5

the available types are:
- `int`
- `float`
- `string`
- `bool`
- `arr`
- `struct` - learn more about in [docs/structs.md](docs/structs.md)