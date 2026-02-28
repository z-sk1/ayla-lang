# Break and Continue

### *Take a break take a kitkat*

break has been renamed to `kitkat`, which exits a loop

and continue has been renamed to `next`, which skips to the next iteration

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

We can use `kitkat` to exit this loop early, for example when `i` reaches 3

```ayla
four i := range 5 {
    ayla i == 3 {
        kitkat
    }
    
    putln(i)
}
```

> output:
```
0
1
2
```

now the loop exits at the 3rd iteration so it only prints the first 3 numbers

We can also use `next` to skip to the next iteration, lets also use it when `i` reaches 3

```ayla
four i := range 5 {
    ayla i == 3 {
        next
    }
    
    putln(i)
}
```
> output:
```
0
1
3
4
```

here the loop skips over the 3rd iteration, so it doesnt print `2`