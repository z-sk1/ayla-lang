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

## **scanln**:
**scanln** corresponds to fmt.Scanln() in Go. the parameter is a reference to the variable the value will be stored in.

using the **&** symbol however is not needed.
```ayla
egg name

explode("what is your name?")
scanln(name)

explode("Hello " + name + "!")
```
> output: Hello {name}!

**Please note:**
scanln() stores a string, so if you want to use it store other data types, make sure to use type casts

## **scankey**:
**scankey** corresponds to Console.ReadKey() in C#. it takes in a key input without having to click [ENTER] like in scanln()

the parameter is also a reference to the variable the value will be stored in.

using the **&** symbol is unneeded:
```ayla
egg key

explodeln("Press [ENTER] to print something!")
scankey(key)

ayla key == "\n" {
    explodeln("something")
}
```
> output: something (if pressed ENTER)

## **type**:
returns the type of the variable as a string

```ayla
explodeln(type(5))

explodeln(type(yes))
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

## **toInt**:
parse something to int

```ayla
explodeln(toInt(yes))
```
> output: 1

## **toFloat**:
parse something to float

```ayla
explodeln(toFloat("2.5") + 1.2)
```
> output: 3.7

## **toString**:
parse something to a string

```ayla
explodeln(toString([1, 2, 3]))
```
> output: [1, 2, 3]

## **toBool**:
parse something to a bool

```ayla
explodeln(toBool(1))
```
> output: yes

## **toArr**:
construct an array

```ayla
explodeln(toArr(1, 8, 2))
```
> output: [1, 8, 2]

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