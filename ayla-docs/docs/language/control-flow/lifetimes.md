# Lifetimes

this is a gimmicky very experimental feature which allows you to determine a variable's lifespan

for example in this snippet
```ayla
egg x<2> = 5
```

the variable x will exist for 2 lines then will delete itself

```ayla
egg x<2> = 5

putln(x) // 5
putln(x) // 5
putln(x) // error
```
> output:
```
5
5
runtime error at 6:7: undefined variable: x
```

you can also use it with multi assignment and type annotation like so
```ayla
egg a, b <3> int = 4, 2

putln(a) // 4
putln(b) // 2
putln(a) // 4
putln(b) // error
```
> output:
```
4
2
4
runtime error at 7:7: undefined variable: b
```

## important note
lifetimes tick based on statements executed, not on uses of that specific variable

Example:
```ayla
egg x <2> = 1

putln(5) // 5
putln(2) // 2
putln(x) // error 
```
> output:
```
5
2
runtime error at 6:7: undefined variable: x
```

this is because you have already used 2 statements so the variable x has already *poofed*