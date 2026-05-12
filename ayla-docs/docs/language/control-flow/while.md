# While Loop

A while loop repeatedly runs a block of code as long as a condition has a `truthy value`.

```ayla
say i = 0

while i < 5 {
    putln(i)
    i = i + 1
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

## infinite loops

since while loops only depend if the condition has a `truthy value`, you can use the constant `yes`, to make a infinite loop like this

```ayla
while yes {
    putln("forever")
}
```

> output:
```
forever
forever
forever
...
```
