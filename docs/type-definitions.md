# type definitions

## named types
named types are a completely new type, that have a native primitive type underneath.

here, `Age` has an underlying type of `int`
```ayla
type Age int
```

but you cant assign `int` to it.

```ayla
type Age int

egg a Age = 3
```
> output: runtime error at 3:4: type mismatch: 'int' assigned to 'Age'

this is because it needs to be casted to `Age`, since the variable `a` is type `Age` not int

```ayla 
type Age int

egg a Age = Age(3)

explode(a)
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

explodeln(x)
```
> output: 5

the available types are:
- `int`
- `float`
- `string`
- `bool`
- `arr`
- `struct` - learn more about in [docs/structs.md](docs/structs.md)