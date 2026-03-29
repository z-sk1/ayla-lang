# Structs
A `struct` is a custom type that groups related values together under named fields.

## syntax

a `struct` type has this syntax:
```ayla
struct{}
```

to declare one use a type statement:
```ayla
type Person struct {
    Name string
}
```

## typed fields

you can put fields in it, then put the corresponding type after it

```ayla
struct {
    X int
    Y int
}
```

## struct literals

to make a literal just add another pair of braces and initalise the fields

```ayla
struct{X int}{X: 12}
```

## accessing fields

if you have a variable which has a `struct` value you can access its fields using the `.` operator

```ayla
egg pos = struct{
    X int
    Y int
}{
    X: 12,
    Y: 4,
}

putln(pos.X)
```
> output:
```
12
```

## zero value
the `zero value` of a `struct` is just a struct with the default value of every field inside it

for example,
```ayla
type Person struct {
    Name string
}
```

the `default value` for `Person` would be:
```ayla
Person{Name: ""}
```