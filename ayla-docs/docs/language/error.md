# Errors as Values

the `error` type represents a failure or exceptional condition

it is commonly returned from functions to indicate that something went wrong.

The `zero value` of `error` is `nil`, meaning no error occurred.

## creating an Error
you can create an `error` using:
```ayla
error("Something went wrong")
```

```ayla
egg err error = error("This is an error")

ayla err != nil {
    putln(err)
}
```
> output:
```
runtime error: This is an error
```

## returning errors from functions
errors are commonly used as a second return value:

```ayla
fun parse(x int) (string, error) {
    if x <= 5 {
        back toString(x), nil
    }

    back "", error("Failed to parse")
}
```

```ayla
x, err := parse(7)

ayla err != nil {
    putln(err)
} elen {
    putln(x)
}
```

## checking for errors
Always check if the error is not nil:

```ayla

ayla err != nil {
    // handle error
}
```
If err is nil, the operation succeeded.

## Important

Creating or returning an error does not automatically stop the program

You must explicitly handle it.
