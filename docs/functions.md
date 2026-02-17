# functions
to declare a function use `fun`

declare them and call them like this

```ayla
fun hi() {
    putln("hi")
}

hi()
```
> output: hi

return has been renamed to `back`, haha

```ayla
fun add(x, y) {
    back x + y
}

put(add(5, 7))
```
output: 12

you **can** have a designated return type like this
```ayla
fun add(x, y) (int) {
    back x + y
}

putln(add(4, 2))
```
> output: 6

ayla also supports multiple return values
```ayla
fun operation(x, y) (int, int) {
    back x + y, x - y
}

putln(operation(4, 5))
```
> output: 9 -1

# methods
methods are functions that are attached to a type

declare them and call them like this

you use the dot `.` syntax, similarily to members
```ayla
type Person struct{}

fun (p Person) greet() {
    putln("hello")
}

x := Person{}

x.greet()
```
> output: hello

you can also use member expressions inside the method like this
```ayla
type Person struct {
    Name string
}

fun (p Person) greet() {
    putln("Hello ${p.Name}")
}

x := Person{
    Name: "Ziad"
}

x.greet()
```
> output: Hello Ziad

also exactly like functions they allow type annotations and multiple return types
```ayla
type Person struct {
    Name string
}

fun (p Person) greet(age int) {
    putln("Hi ${p.Name} you are {age}")
}

x := Person{
    Name: "Ziad"
}

x.greet(13)
```
> output: Hi Ziad you are 13

```ayla
type Person struct {
    Name string
    Age int
}

fun (p Person) getInfo() (string, int) {
    return p.Name, p.Age
}

x := Person{
    Name: "Ziad",
    Age: 13,
}

name, age := x.getInfo()

putln(name, age)
```
> output: Ziad 13

## disclaimer
points and refs are not yet a feature so you can only make the receiver a value copy

which means that it will only adjust what happens inside the function
```ayla
type Person struct {
    Name string 
}

fun (p Person) switchName(name string) {
    p.Name = name
    putln(p.Name)
}

x := Person{
    Name: "Zi"
}

x.switchName("Ziad") // Ziad

putln(x.Name) // Zi
```
> output:
```
Ziad
Zi
```

since that would usually require something like `(p *Person)`

### fun note!
methods work like normal functions under the hood, but they make their receivers the first parameter

so a method like this
```ayla
fun (p Person) greet(age int) {
    putln("Hi ${p.Name} you are ${age}")
}
```

would correspond to a function like this
```ayla
fun greet(p Person, age int) {
    putln("Hi ${p.Name} you are ${age}")
}
```
