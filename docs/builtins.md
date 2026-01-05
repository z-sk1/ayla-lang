## **explode**:
prints values to the console, other, pettier, langs call it "print"

```ayla
explode("Hi")
explode("ayla")
```
> output: Hi ayla

## **explodeln**:
same as explode but with a '\n' instead 

```ayla
explodeln("Hi")
explodeln("ayla")
```
> output: 
```
Hi
ayla
```

## **tsaln**:
**tsaln** corresponds to fmt.Scanln() in Go. the parameter is a reference to the variable the value will be stored in.

using the **&** symbol however is not needed.
```ayla
egg name

explode("what is your name?")
tsaln(name)

explode("Hello " + name + "!")
```
> output: Hello {name}!

**Please note:**
tsaln() stores a string, so if you want to use it store other data types, make sure to use type casts

## **type casts**
ayla lang supports 4 different casts.

- string()
```ayla
explode(string(4) + 2)
```
> output: 42

- int()
```ayla
egg num1
egg num2

explode("num 1?")
tsaln(num1)

explode("num 2?")
tsaln(num2)

explode(int(num1) + int(num2))
```
> input: 5, 4

> output: 9

- float()
```ayla
egg num1
egg num2

explode("num 1?")
tsaln(num1)

explode("num 2?")
tsaln(num2)

explode(float(num1) + float(num2))
```
> input: 5.5, 4.5

> output: 10.0

- bool()
```ayla
explode(bool(5))

explode(bool(0))

explode(bool(-3))
```
> output: yes, no, yes

## **type**:
returns the type of the variable as a string

```ayla
explode(type(5))

explode(type(yes))
```
> output: int, bool

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

## **push**:
append a value to end of an array

```ayla
egg arr = [1, 2, 3]

push(arr, 4)

explode(arr)
```
> output: [1, 2, 3, 4]

## **pop**:
remove, and return the last element of an array

```ayla 
egg arr = [1, 2, 3, 4]

egg val = pop(arr)

explode(val)
```
> output: 4

## **insert**:
insert a value into an array at a certain index

```ayla
egg arr = [1, 2, 4]

insert(arr, 2, 3)

explode(arr)
```
> output: [1, 2, 3, 4]

## **remove**:
remove and return an element at an index

```ayla
egg arr = [1, 2, 3, 5, 4]

egg odd = remove(arr, 3)

explode("Odd number: " + odd)
```
> output: Odd number: 5

## **clear**:
remove all elements of an array

```ayla
egg arr = [1, 2, 3, 4]

clear(arr)
explode(arr)
```
> output: []

## **wait**:
takes in a duration of milliseconds then pauses the program for that duration

```ayla
explodeln("Finishing task...")
wait(2000) // 2 seconds
explodeln("done!")
```
> output:
```
Finishing task...
done!
```

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