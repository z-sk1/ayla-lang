# ayla lang
ayla lang is a small interpreted language written in go, designed to make you forget everything

*Because fuck you.* - Linus Torvalds

# about

## our team
- **Me: z-sk1, Co-Owner**
- **and Mregg55, Co-Owner (link: https://github.com/mregg55)**

## vs code extension
https://marketplace.visualstudio.com/items?itemName=z-sk1.ayla

this will add syntax highlighting

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

```
egg arr = [0, 1, 2, 3]

explode(arr)
```
> output: [0, 1, 2, 3]

you can also index into an array, like normal

```
egg arr = [1, 2, 5]

explode(arr[2])
```
> output: 5

and you can reassign a specific index
```
egg arr = ["hello", 1]

arr[1] = "world"

explode(arr)
```
> output: [hello, world]

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


### miscellaneous
version:

```ayla
ayla --version
```

help:

```ayla
ayla --help
```