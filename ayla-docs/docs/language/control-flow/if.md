# If Statement

an if statement allows your program to make decisions

it runs a block of code only if a condition is true

use the `ayla` keyword for if statements

```ayla
egg age = 13

ayla age >= 13 {
    putln("You are a teenager")
}
```
> output: 
```
You are a teenager
```

This is because the boolean expression age (13) >= 13 evaluates to `yes`, which is a `truthy` value

learn about booleans and booleans expressions [here](../booleans.md)

## else
you can provide an alternative block using else

the keyword for else is `elen`

```ayla
egg age = 10

ayla age >= 13 {
    putln("Teenager")
} elen {
    putln("Not a teenager")
}
```
> output:
```
Not a teenager
```

## else if
you can chain multiple conditions

```ayla
egg score = 85

ayla score >= 90 {
    putln("A")
} elen ayla score >= 80 {
    putln("B")
} elen {
    putln("C or lower")
}
```
> output:
```
B
```

the conditions are checked in order, and the first `truthy` condition runs

## nested If Statements
You can place an if inside another if:

```ayla
egg age = 15
egg hasID = yes

ayla age >= 13 {
    ayla hasID {
        putln("Entry allowed")
    }
}
```
> output:
```
Entry allowed
```