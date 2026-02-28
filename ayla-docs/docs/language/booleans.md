# Booleans

a boolean represents a truth value.

in Ayla, a boolean can be either:
- `yes` (true)
- `no` (false)


## boolean expressions
booleans are often produced by comparisons:

```ayla
egg x = 5

putln(x > 3)   // yes
putln(x == 10) // no
```
> output:
```
yes
no
```

comparison operators include:
- `==` (equal)
- `!=` (not equal)
- `>` (greater than)
- `<` (less than)
- `>=` (greater than or equal)
- `<=` (less than or equal)

## logical operators
You can combine boolean values:

```ayla
egg a = yes
egg b = no

putln(a && b)  // no
putln(a || b)  // yes
putln(!a)      // no
```
> output:
```
no
yes
no
```

where `&&` is AND, `||` is OR, and `!` is NOT
