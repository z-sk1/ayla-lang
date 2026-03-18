# Enums

Enums define a set of named variants under a single type

## declaration
declare them like this
```ayla
enum Color {
  Red
  Blue
  Green
}
```
here each variant belongs to the `Color` type

## using variants
you can use enum variants as values using member expressions like this
```ayla
egg c = Color.Red
```

since enums define a new type you can use them in type annotation
```ayla
egg c Color = Color.Blue
```

## enums as types
since Enums are first-class types you can use them in things like function parameters too
```ayla
fun foo(c Color) {
  // ...
}

foo(Color.Blue)
```

## some notes:
- Enum variant names must be unique within the `enum`
- Enums cannot be redeclared
- Accessing an unknown variant is a `runtime error`
