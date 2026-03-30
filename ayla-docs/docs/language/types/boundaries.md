# Boundaries

boundaries are a feature of `int` and `float` types 

they allow you to specify a range for numerical values

## syntax
```ayla
int<start..end>
float<start..end>
```
`start` is the minimum allowed value.
`end` is the maximum allowed value.

## example
```ayla
egg age int<0..120> = 25
egg temperature float<-273.15..10000>
```
Here:
- `age` cannot be less than 0 or greater than 120
- `temperature` cannot go below absolute zero

## runtime behavior
ff a value is assigned outside the boundary range, the program will produce an error.

Example:
```ayla
age int<0..120> = 200
```
> output: 
```
runtime error at 1:4: value 200 above maximum 120
```

## using boundaries in function parameters
```ayla
fun setVolume(level int<0..100>) {
    putln(level)
}
```
this guarantees the function only receives valid values

## notes
- boundaries work with both `int` and `float`
- the range is inclusive, meaning the `start` and `end` values are allowed
- boundaries are checked when assigning values
