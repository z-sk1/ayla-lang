# For Loop
A loop allows you to repeat a block of code multiple times.

the for loop has been renamed to `four` loop

## C-Style
this is a classic c-style for loop
```ayla
four egg i = 0; i < 5; i = i + 1 {
    put(i) 
}
```
> output: 0 1 2 3 4

### How it works
a C-style loop has three parts:

```
initialisation ; condition ; update
```

`Initialisation` = `egg i = 0`
creates a counter variable starting at 0.

`Condition` = `i < 5`
The loop runs as long as this is true.

`Update` = `i = i + 1`
Runs after each iteration and increases i.

So this loop:

- Starts at 0
- Runs while `i` is less than 5
- Increases `i` each time
- Stops when `i` becomes 5

## Range style
but you can also do a for loop with `range` to iterate over:
- maps
- arrays
- slices
- strings
- ints

### arrays:
```ayla
x := []int{1, 2, 3}

four i, v := range x {
    putln(v)
}
```
> output:
```
1
2
3
```

### maps:
```ayla
x := map[string]int{"a": 1, "b": 2}

four k, v := range x {
    putln(k)
}
```

> output: 
```
a
b
```

### strings:
```ayla
x := "hiya"

four i, v := range x {
    putln(v)
}
```
> output:
```
h
i
y
a
```

### ints:
this is used as a repeat
```ayla
four i := range 5 {
    putln(i)
}
```
> output:
```
0
1
2
3
4
```

you can also use `_` to discard a variable like this
```ayla
x := []int{1, 2, 3}

four _, v := range x {
    putln(v)
}
```
> output:
```
1
2
3
```
