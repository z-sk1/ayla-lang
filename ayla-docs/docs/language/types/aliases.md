# Aliases
Unlike named types, which create a completely new type with an underlying base type, `aliases` are simply a new name for an existing type.

**Aliases do not create a distinct type.**

You define an `alias` using the `type` keyword with `=`:

```ayla
type Age = int
```

here, `Age` is exactly equivalent to `int`.

so because `Age` is just another name for `int`, they are fully interchangeable:

```ayla
type Age = int

egg a Age = 5
egg y int = a
putln(y)
```
> output: 5

no conversion is required because Age and int are the same type.

## alias vs named Type
Compare:
```ayla
type Age int
```

this creates a new type based on int.

Now this would require explicit conversion:

```ayla 
egg a Age = 5
egg y int = int(a)  // required
```

but with an alias:
```ayla
type Age = int

egg a Age - 5
egg y int = a
```

no conversion is needed.
