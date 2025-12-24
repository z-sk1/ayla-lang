# about

## our team
- **Me: z-sk1, Co-Owner**
- **and Mregg55, Co-Owner (link: https://github.com/mregg55)**



# the features

## declaration and assignment
to declare a normal mutable, reassignable variable use **egg** 
```
egg x = "wowie"
```

to declare a constant, use **rock**

```
rock x = "i will never change"
```

## semicolon
semicolons are optional! put them if you want, or leave them out if you're more comfortable with that
```
egg x = 5;
```
this is totally fine

```
egg x = 5
```
also valid

## booleans
booleans can be either yes or no

```
egg foo = yes

ayla foo {
    explode("foo is yes")
} elen {
    explode("foo is no")
}
```
> output: foo is yes

## if/else if/else

in ayla-lang, if has been renamed to **ayla**, and else renamed to **elen**. therefore else if has been aptly renamed to **elen ayla**.


```
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

```
four egg i = 0; i < 5; i = i + 1 {
    explode(i) 
}
```
> output: 1 2 3 4 5

### why loop
the while loop has been renamed to **why** loop, for convenience

no brackets here either

:>

```
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

```
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

```
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

```
blueprint add(x, y) {
    back x + y
}

explode(add(5, 7))
```
output: 12

you cant have a designated return type like this, yet
```
func test() int {
    return something
}
```

so uh have fun with that :-)

## built in functions!

## **len**:
### supports strings and arrays

```
egg arr = [1, 2, 3, 4]

explode(len(arr))
```
> output: 4

```
egg str = "ayla wow"

explode(len(str))
```
> output: 8

## **randi**:
returns a random integer

### if zero args are present will return either 0 or 1

```
explode(randi())
```
> output: 0 or 1

### if there is 1 arg, it will return a random number between 0 and the arg
```
explode(randi(5))
```
> output: 0 - 5

### if there are 2 args, it will return a random number between the first and second arg *(min, max)*
```
explode(randi(5, 10))
```
> output: 5 - 10

## **randf**

### all the same features as randi, but for floats

## type casting

as of now, you can cast int(), string(), and float()

```
egg foo = "12"

explode(int(foo) + 5)
```
> output: 17

## arrays
to initialise an array use square brackets: **[]**

```
egg arr = [0, 1, 2, 3]

explode(arr)
```
> output: [0 1 2 3]

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
> output: [hello world]

## cli tooling and running scripts

### windows instructions
to use the cli, please go to the releases tab and download the zip file.

extract the zip, and put the exe file in a easy to access place, like C:\ayla

put the file path in your PATH found in your System Environment Variables

there isnt a REPL currently, so make sure to put **ayla** infront of every cmd

### running

to run a script do:

```
ayla run [--debug] [--timed] <file>
```
> --debug will give debug info like ast, and tokens

> --timed will time how long your program takes


### miscellaneous
version:

```
ayla --version
```

help:

```
ayla --help
```