# Modules & Importing
ayla organizes code using `modules`. Each file represents a `module`, and `modules` can import definitions from other `modules`.

## module mame
the module name is the file name

Example file:
```
math.ayla
```
The module name becomes:
```ayla
math
```

you can then import it from another file.

## importing modules
Use the `import` statement to load another `module`.
```ayla
import math
```
after importing, you can access exported members from that `module`.

Example:
```ayla
import math

result := math.Min(5, 6)
```

## import resolution
when Ayla resolves an import, it searches for the module in the following order:

### Global library directory
```
~/.ayla/lib
```
(or the equivalent AppData directory on Windows)

### Project library directory
```
./lib
```
### Current project directory
```
./
```

the first matching file is used.
