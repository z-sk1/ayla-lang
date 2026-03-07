# Custom Types

ayla allows you to define custom types using the `type` keyword

```ayla
type Foo int
```
this creates a new type `Foo` with an underlying type of `int`

## casting

Ayla also supports type casting for custom types
```ayla
type Foo int

egg a = 5
egg b = Foo(a)
```

here `a` is an `int` and `b` is a `Foo`

So even though `Foo` is based on `int`, they are considered distinct types

For example, this will produce a `type error` due to the fact that they are not the same type

```ayla
type Foo int

egg x Foo = 5
egg y int = x
```
> output: runtime error: type mismatch: 'Foo' assigned to 'int'

to convert back just do:
```ayla
egg y int = int(x)
```

## composites

you can also make custom types as composites

this includes things like slices, arrays, structs, and maps

Lets make a `Names` slice type and use it!
```ayla
type Names []string

egg x = Names{"Ziad", "Ayla", "Elen"}
```
This works because the composite literal,

```ayla
[]string{"Ziad", "Ayla", "Elen"}
```

got converted to Names like this

```ayla
Names([]string{"Ziad", "Ayla", "Elen"})
```

# Aliases