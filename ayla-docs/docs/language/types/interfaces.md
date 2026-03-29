# Interfaces

An `interface` is a type which defines a set of methods, and any type which implements those methods will satisfy the `interface`

This means that no explicit definition is need

## syntax

an `interface` type has this syntax
```ayla
interface {
    Greet() (string)
}
```

to declare one use a type statement:
```ayla
type Greeter interface {
    Greet() (string)
}
```

## implementing an interface
in ayla you dont need to use an `implements` keyword, if your type has the methods then it automatically satisfies the `interface`

```ayla
type Greeter interface {
    Greet() (string)
}

type Person struct {
    Name string
    Age  int
}

fun (p Person) Greet() (string) {
    back "Hi I'm {p.Name} and I am {p.Age} years old"
}
```
now `Person` implements the `Greeter` `interface`, so it can be used as a `Greeter` '

## using an interface 
```ayla
fun greet(g Greeter) {
    putln(g.Greet())
}

p := Person{Name: "Ziad", Age: 13}
greet(p)
```
> output:
```
Hi I'm Ziad and I am 13 years old 
```

## empty interfaces
an empty `interface` has no methods, meaning every type satisfies it. in Ayla, the `thing` type is an `alias` for an empty `interface`:
```ayla
egg x interface{} = 4
egg y thing = "hello"
egg z thing = yes
```

`thing` is useful when you need to store or pass values of unknown or mixed types.

## type assertions
since a `thing` (or any interface) value could be any type, you need to `assert` it before using it in operations:

```ayla
egg x thing = 42
putln(x.(int) + 1)
```

> output:
```
43
```

asserting the wrong type causes a runtime error:

```ayla
egg x thing = "hello"
egg n int = x.(int) // runtime error: type mismatch
```
> output:
```
runtime error at 2:19: type mismatch: 'string' asserted as 'int'
```

## pointer receivers and interfaces
if a method is defined with a pointer receiver, only the pointer to that type implements the interface, not the type itself:

```ayla
type Greeter interface {
    Greet() (string)
}

type Person struct {
    Name string
}

fun (p *Person) Greet() (string) {
    back "Hi I'm ${p.Name}"
}
```

here `*Person` implements `Greeter`, but `Person` does not, so you must use `&Person{}`:
```ayla
p := &Person{Name: "Ziad"}  // correct, *Person implements Greeter
greet(p)

p2 := Person{Name: "Ziad"}  // wrong, Person does not implement Greeter
greet(p2)                    // type error
```
> output:
```
runtime error at 21:9: param 'g' of type 'Person' does not implement 'Greeter' (missing method 'Greet')
```

## interface with multiple methods

an interface can require as many methods as needed:

```ayla
type Shape interface {
    Area()      (float)
    Perimeter() (float)
}

type Rect struct {
    Width  float
    Height float
}

fun (r Rect) Area() (float) {
    back r.Width * r.Height
}

fun (r Rect) Perimeter() (float) {
    back 2 * (r.Width + r.Height)
}
```
`Rect` satisfies `Shape` because it implements both `Area` and `Perimeter`
