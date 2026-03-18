# Switch-Case Statement
a switch-case statement allows your program to choose between multiple possible cases based on a value

switch has been renamed to `choose`
and case to `when`
and default to `otherwise`

Here, it carries out instructions based on what integer value `x` will have
```ayla
egg x = 2

choose x {
    when 2 {
        put("x is 2")
    }

    when 3 {
        put("x is 3")
    }

    otherwise {
        put("x is neither 2 or 3")
    }
}
```
> output: x is 2

you can also use conditionals in the switch expression, like this

```ayla
egg x = 5

choose x < 10 {
    when yes {
        put("x is less than 10")
    }

    otherwise {
        put("x is more than 10")
    }
}
```


you can also implement conditionals into case expressions by making the switch expression a boolean value
```ayla
egg x = 5

choose yes {
    when x < 10 {
        put("x is less than 10")
    }

    otherwise {
        put("x is more than 10")
    }
}
```