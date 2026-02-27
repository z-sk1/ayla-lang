## **put**:
prints values to the console, other, pettier, langs call it "print"

```ayla
put("Hi")
put("ayla")
```
> output: Hi ayla

## **putln**:
same as put but with a '\n' instead 

```ayla
putln("Hi")
putln("ayla")
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

put("what is your name?")
scanln(name)

put("Hello " + name + "!")
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

putln("Press [ENTER] to print something!")
scankey(key)

ayla key == "\n" {
    putln("something")
}
```
> output: something (if pressed ENTER)

## **typeof**:
returns the type of the variable as a string

```ayla
put(typeof(5))

put(typeof(yes))
```
> output: int bool

## **len**:
### supports strings and arrays

```ayla
egg arr = []int{1, 2, 3, 4}

put(len(arr))
```
> output: 4

```ayla
egg str = "ayla wow"

put(len(str))
```
> output: 8

## **toInt**:
parse something to int

```ayla
putln(toInt(yes))
```
> output: 1

## **toFloat**:
parse something to float

```ayla
putln(toFloat("2.5") + 1.2)
```
> output: 3.7

## **toString**:
parse something to a string

```ayla
putln(toString([]int{1, 2, 3}))
```
> output: [1, 2, 3]

## **toBool**:
parse something to a bool

```ayla
putln(toBool(1))
```
> output: yes

## **wait**:
takes in a duration of milliseconds then pauses the program for that duration

```ayla
putln("Finishing task...")
wait(2000) // 2 seconds
putln("done!")
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
put(randi())
```
> output: 0 or 1

### if there is 1 arg, it will return a random number between 0 and the arg
```
put(randi(5))
```
> output: 0 - 5

### if there are 2 args, it will return a random number between the first and second arg *(min, max)*
```
put(randi(5, 10))
```
> output: 5 - 10

## **randf**

### all the same features as randi, but for floats