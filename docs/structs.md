# structs
there are two types of structs in ayla lang, typed structs, and anonymous structs

## typed structs 
firstly, typed structs require you to define a type of struct like this

```ayla
type Person struct {
    Name string
    Age int
}
```

and then use it to store data in variables like this

```ayla 
egg p = Person{
    Name: "Ziad",
    Age: 13
}
```

then use the `.` symbol to access the fields inside the struct
```ayla
egg name = p.Name
egg age = p.Age

explode("${name} is ${age} years old")
```
> output: Ziad is 13 years old

however, if you use the wrong type then you come across a `Runtime error`
```ayla
type Operation struct {
    Left int
    Right int
}

egg o = Operation{
    Left: "5",
    Right: 3
}
```
> output: runtime error at 6:19: field 'Left' type string should be int

same thing for member expressions and assignment
```ayla
type Person struct {
    Name string
    Age int
}

egg p = Person{
    Name: "Ziad",
    Age: 13
}

p.Age = "13"
```
> output: runtime error at 11:13: field 'Age' type string should be int

if you use an unknown field not declared in the type struct you will also encounted a `Runtime error`
```ayla
type Person struct {
    Name string
    Age int
}

egg p = Person{
    Name: "Ziad",
    Age: 13,
    Extra: "extra field"
}
```
> output: runtime error at 6:16: unknown field 'Extra' in struct Person

## anonymous structs
then, anonymous structs dont require you to define a type


for anonymous structs, since there is no struct type, just use the `struct` keyword to denote an anonymous struct


you can just initialise them with `struct {}`, in a similar way to arrays

```ayla
egg Operation = struct {
    Left: 5,
    Right: 4
}

explode("${Operation.Left} + ${Operation.Right} = ${Operation.Left + Operation.Right})
```
> output: 5 + 4 = 9

you can also use member expressions and assignment using the `.` symbol
```ayla
egg oper = strict {
    Left: 4,
    Right: 5,
}

oper.Right = 10

explode(oper.Right + oper.Left)
```
> output: 14

you will also come across `Runtime errors` using anonymous structs in the same way as typed structs

this happens if you use a different type compared to the one you declared in the field

```ayla
egg oper = struct {
    Left: 5, // int
    Right: 10
}

oper.Left = "hi" // string
```
> output: runtime error at 6:17: field 'Left' type string should be int