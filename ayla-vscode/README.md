# ayla extension

syntax highlighting and ayla file icons

# ayla lang
ayla lang is a small interpreted language written in go, designed to make you forget everything

*Because fuck you.* - Linus Torvalds

# about

## our team
- **Me: z-sk1, Co-Owner**
- **and Mregg55, Co-Owner (link: https://github.com/mregg55)**

# the features

## declaration and assignment
to declare a normal mutable, reassignable variable use **egg** 
```ayla
egg x = "wowie"
```

variables can be declared without an initial value, like so.
```ayla
egg x
```
they default to `nil`

to declare a constant, use **rock**

```ayla
rock x = "i will never change"
```

constants cannot be declared without an intitial value.
```ayla
rock x
```
> output: Runtime error at 1:5: const x must be initialised

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
booleans can be either yes or no

```ayla
egg foo = yes

ayla foo {
    explode("foo is yes")
} elen {
    explode("foo is no")
}
```
> output: foo is yes

## string concatenation
you can concatenate strings using the **+** operator.

```ayla
egg a = "hello "
egg b = "world"

explode(a + b)
```
> output: hello world

you can also concatenate strings with other types by casting.
```ayla 
explode(string(4) + 2)
```
> output: 42

## string interpolation
you can also interpolate strings using **${}**

> unlike JavaScript, you just use the normal quotation marks, " ", not ` `

```ayla
egg rand = randi(10)

explode("Random number: ${rand}")
```
> output: 0 - 10

## if/else if/else

in ayla-lang, if has been renamed to **ayla**, and else renamed to **elen**. therefore else if has been aptly renamed to **elen ayla**.


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
the for loop has been renamed to **four** loop, for convenience

oh yea also no brackets

*for convenience*

```ayla
four egg i = 0; i < 5; i = i + 1 {
    explode(i) 
}
```
> output: 1 2 3 4 5

### why loop
the while loop has been renamed to **why** loop, for convenience

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

because we are so nice, we renamed break to **kitkat** so it sticks in your memory

oh yea we also renamed continue to **next**

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
switch has been renamed to **decide**
and case to **when**
and default to **otherwise**

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

nuh uh now theyre called blueprints

return has been renamed to back, haha

```ayla
blueprint add(x, y) {
    back x + y
}

explode(add(5, 7))
```
output: 12

you cant have a designated return type like this, yet
```ayla
func test() int {
    return something
}
```

so uh have fun with that :-)

## arrays
to initialise an array use square brackets: **[]**

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

## cli tooling and running scripts

### windows instructions
to use the cli, please go to the releases tab and download the zip file.

extract the zip, and put the exe file in a easy to access place, like C:\ayla

put the file path in your PATH found in your System Environment Variables

there isnt a REPL currently, so make sure to put **ayla** infront of every cmd

### running

to run a script do:

```bash
ayla run [--debug] [--timed] <file>
```
> --debug will give debug info like ast, and tokens

> --timed will time how long your program takes

```bash
ayla run test.ayl
```
and also for the other extension
```bash
ayla run test.ayla
```

you can also do it without putting a file extension
```bash
ayla run test
```
this will first try appending .ayla, then if not found it will try appending .ayl

if test.(ayl/ayla) does not exist then ayla CLI will throw an error:
```bash
file not found: test.ayla (.ayla or .ayl) 
```


### miscellaneous
version:

```bash
ayla --version
```

help:

```bash
ayla --help
```