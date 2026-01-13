# ayla lang
<img width="512" height="512" alt="ayla-512" src="https://github.com/user-attachments/assets/1a266fdd-0d0d-4f95-83fa-1fd5d7bca0f9" />

ayla lang is a small interpreted language written in go, designed to make you forget everything

*Because fuck you.* - Linus Torvalds

# about

## our team
- **Me: z-sk1, Co-Owner**
- **and Mregg55, Co-Owner (link: https://github.com/mregg55)**

## vs code extension
https://marketplace.visualstudio.com/items?itemName=z-sk1.ayla

this will add syntax highlighting

# Installation and Usage
See [INSTRUCTIONS.md](./INSTRUCTIONS.md) for full step-by-step instructions for macOS and Windows.

---

# the features

## declaration and assignment
to declare a normal mutable, reassignable variable use `egg`
```ayla
egg x = "wowie"
```

variables can be declared without an initial value, like so.
```ayla
egg x
```
they default to `nil`

to declare a constant, use `rock`

```ayla
rock x = "i will never change"
```

constants cannot be declared without an intitial value.
```ayla
rock x
```
> output: Runtime error at 1:5: const x must be initialised

you can also use type annotation for both `egg` and `rock`

the available types are:
- `int` 
- `float`
- `string`
- `bool`

```ayla 
egg x int = 3

explodeln(x)
```
> output: 3

and you will also come across `Runtime errors` if you address the wrong type
```ayla
rock x string = 5
```
> output: runtime error at 1:5: type mismatch: 'int' assigned to a 'string'

## multi-declaration and multi-assignment
you can also assign and declare multiple variables to a `tuple`

```ayla
rock a, b = 4, 2

explodeln("${a} ${b}")
```
> output: 4 2

you can also just declare them without an inital value like normal
```ayla
egg a, b
```

you can do type annotation, but not for every variable, the type at the end dictates all the other's types like so
```ayla
egg a, b, c int

explodeln(type(a), type(b), type(c))

explodeln(a)
```
> output:
```
int
int
int
0
```

and also like single declaration, using multi constant declaration you must initialise it
```ayla
rock a, b
```
> output: runtime error at 1:5: constants, a, b, must be initialised

same for statically typed multi constants
```ayla
rock a, b int
```
> output: runtime error at 1:5: constants, a, b, must be initialised

same principles for assignment
```ayla
egg a, b

a, b = 4, 2

explodeln("${a} ${b}")
```
> output: 4 2

you can also assign and declare variables to a function with multiple return values
```ayla
fun operation(x int, y int) (int, int) {
    back x + y, x - y
}

egg a, b int = operation(5, 3)

explodeln(a, b)
```
> output:
```
8
2
```

## semicolon
semicolons are optional! put them if you want, or leave them out if you're more comfortable with that
```ayla
egg x = 5;
```
this is totally fine

```ayla
egg x = 5
```
also valid

## block scope
in ayla, every block has its own **Environment**.
```ayla
ayla yes {
    egg x = 2 // define inside if statement
}

explode(x) // error
```
> output: runtime error at 5:10: undefined variable: x

if the lower scope cant find the variable, it will look for it in the parent environment
```ayla
egg x = 4

ayla yes {
    x = 2
}

explode(x)
```
> output: 2


but you can also define a variable with the same name in the child environment, and it wont affect the one above
```ayla
egg x = 5

ayla yes {
    explodeln(x) // 5

    x = 3

    explodeln(x) // 3

    egg x = 7

    explodeln(x) // 7
}

explodeln(x) // 3, because was assigned 3 in child
```
> output:
```
5
3
7
```

## comments
you can use both single line and multiline comments

`//` is for single line comments:

```ayla
egg x = 5 // this is a comment

// and this is also a comment
```

`/*` open and `*/` close for multiline comments:
```ayla
egg x = 5

/* this is
a really 
big comment */
```

## booleans
for booleans, it is recommended to use the constants `yes` and `no`
```ayla
egg foo = yes

ayla foo {
    explode("foo is yes")
} elen {
    explode("foo is no")
}
```
> output: foo is yes

but, you can also assign them any `string`, `int`, or `float` value

these are the values that assign the boolean to `no`:
- `""`
- `0`
- `0.0`
- `nil`
- `no`

all the other values will give the boolean a `yes` value

```ayla
egg x bool = 42

explodeln(x)
```
> output: yes

```ayla
egg x bool = 0

explodeln(x)
```
> output: no

```ayla
egg x bool = ""

explodeln(x)
```
> output: no

*also with negatives*
```ayla
egg x bool = -2.2

explodeln(x)
```
> output: yes

## string concatenation
you can concatenate strings using the `+` operator.

```ayla
egg a = "hello "
egg b = "world"

explode(a + b)
```
> output: hello world

you can also concatenate strings with other types by casting.
```ayla 
explode(string(4) + string(2))
```
> output: 42

## string interpolation
you can also interpolate strings using `${}`

> unlike JavaScript, you just use the normal quotation marks, "", not ``

```ayla
egg rand = randi(10)

explode("Random number: ${rand}")
```
> output: 0 - 10

## string indexing
you can index into strings almost like arrays

```ayla 
egg text = "Hello"

explodeln(text[0])
```
> output: H

## if/else if/else

in ayla-lang, if has been renamed to `ayla`, and else renamed to `elen`. therefore else if has been aptly renamed to `elen ayla`.


```ayla
egg x = 5

ayla x <= 9 {
    explode("number is single digits")
} elen ayla x >= 10 {
    explode("number is double digits")
}

```

## loops

### four loop
the for loop has been renamed to `four` loop, for convenience

oh yea also no brackets

*for convenience*

```ayla
four egg i = 0; i < 5; i = i + 1 {
    explode(i) 
}
```
> output: 1 2 3 4 5

### why loop
the while loop has been renamed to `why` loop, for convenience

no brackets here either

:>

```ayla
egg i = 0

why i < 7 {
    i = i + 1

    explode(i)
}
```
> output: 1 2 3 4 5 6 7

### kitkat and next
*Take a break, take a kitkat*

because we are so nice, we renamed break to `kitkat` so it sticks in your memory

oh yea we also renamed continue to `next`

```ayla
egg i = 0

why i < 7 {
    i = i + 1

    ayla i == 4 {
        kitkat
    }

    explode(i)
}
```
> output: 1 2 3

```ayla
egg i = 0

why i < 7 {
    i = i + 1

    ayla i == 4 {
        next
    }

    explode(i)
}
```
> output: 1 2 3 5 6 7

## switch-case 
switch has been renamed to `decide`
and case to `when`
and default to `otherwise`

```ayla
egg x = 2

decide x {
    when 2 {
        explode("x is 2")
    }

    when 3 {
        explode("x is 3")
    }

    otherwise {
        explode("x is neither 2 or 3")
    }
}
```
> output: x is 2

you can also use conditionals in the switch expression, like this

```ayla
egg x = 5

decide x < 10 {
    when yes {
        explode("x is less than 10")
    }

    otherwise {
        explode("x is more than 10")
    }
}
```


you can also implement conditionals into case expressions by making the switch expression a boolean value
```ayla
egg x = 5

decide yes {
    when x < 10 {
        explode("x is less than 10")
    }

    otherwise {
        explode("x is more than 10")
    }
}
```

## functions

to declare a function use `fun`

return has been renamed to `back`, haha

```ayla
fun add(x, y) {
    back x + y
}

explode(add(5, 7))
```
output: 12

you **can** have a designated return type like this
```ayla
fun add(x, y) (int) {
    back x + y
}

explodeln(add(4, 2))
```
output: 6

you will encounter a `Runtime error` if you use the wrong type
```ayla
fun add(x, y) (string) {
    back x + y
}

explodeln(add(4, 1))
```
> output: runtime error at 5:14: return 1, expected string, got int

you can also add types to parameters
```ayla
fun add(x int, y int) (int) {
    back x + y
}
```

you will encounter a `Runtime error` as well if you use the wrong type for the parameter
```ayla
fun add(x int, y int) (int) {
    back x + y
}

egg sum = add("4", 2)
```
> output: runtime error at 5:14: paramteter 'x' expected int, got string

## arrays
to initialise an array use square brackets: `[]`

```ayla
egg arr = [0, 1, 2, 3]

explode(arr)
```
> output: [0, 1, 2, 3]

you can also index into an array, like normal

```ayla
egg arr = [1, 2, 5]

explode(arr[2])
```
> output: 5

and you can reassign a specific index
```ayla
egg arr = ["hello", 1]

arr[1] = "world"

explode(arr)
```
> output: [hello, world]

## structs
there are two types of structs in ayla lang, typed structs, and anonymous structs

### typed structs 
firstly, typed structs require you to define a type of struct like this

```ayla
struct Person {
    Name string
    Age int
}
```

and then use it to store data in variables like this

```ayla 
egg p = Person {
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
struct Operation {
    Left int
    Right int
}

egg o = Operation {
    Left: "5",
    Right: 3
}
```
> output: runtime error at 6:19: field 'Left' type string should be int

same thing for member expressions and assignment
```ayla
struct Person {
    Name string
    Age int
}

egg p = Person {
    Name: "Ziad",
    Age: 13
}

p.Age = "13"
```
> output: runtime error at 11:13: field 'Age' type string should be int

if you use an unknown field not declared in the type struct you will also encounted a `Runtime error`
```ayla
struct Person {
    Name string
    Age int
}

egg p = Person {
    Name: "Ziad",
    Age: 13,
    Extra: "extra field"
}
```
> output: runtime error at 6:16: unknown field 'Extra' in struct Person

### anonymous structs
then, anonymous structs dont require you to define a type

you can just initialise them with {}, in a similar way to arrays

```ayla
egg Operation = {
    Left: 5,
    Right: 4
}

explode("${Operation.Left} + ${Operation.Right} = ${Operation.Left + Operation.Right})
```
> output: 5 + 4 = 9

you can also use member expressions and assignment using the `.` symbol
```ayla
egg oper = {
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
egg oper = {
    Left: 5, // int
    Right: 10
}

oper.Left = "hi" // string
```
> output: runtime error at 6:17: field 'Left' type string should be int

## built in functions!
- `explode(...)` – prints values to stdout
- `explodeln(...)` — prints values to stdout and adds '\n' at the end
- `tsaln(x)` – scans console input and stores it in variable
- `bool(x)` – converts a value to boolean
- `string(x)` – converts a value to string
- `int(x)` – converts a value to integer
- `float(x)` – converts a value to float
- `type(x)` – returns type of value as string
- `len(x)` – returns length of arrays or strings
- `push(arr, val)` – append to array
- `pop(arr)` – remove and return last element
- `insert(arr, index, val)` – insert value
- `remove(arr, index)` – remove element at index
- `clear(arr)` – remove all elements
- `wait(ms)` – wait for a duration in milliseconds
- `randi()` or `randi(max)` or `randi(min, max)`
- `randf()` or `randf(max)` or `randf(min, max)`

See [docs/builtins.md](docs/builtins.md) for more about built-in functions.

## runtime errors
error handling for runtime errors
```ayla
rock i = 1

i = 2
```
> Runtime error at 3:2: cannot reassign to const: i

## parse errors
error handling for parse errors
```ayla
ayla {

}
```
> output: parse error at 1:6: missing condition in if (got {)

parse errors will default to (got nothing) if there is nothing after the token
```ayla
egg x =
```
> output: parse error at 1:8: expected expression after '=' (got nothing)