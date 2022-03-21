
# Type matrix

The following matrices display the type relations and inter-type acceptance.

## Assignment

The below matrix shows the use of the assignment operator (:=) to set a variable,
identified by an identifier, of type to the value of either another identifier/function call
or int or float literals. The declared column is the data type of the variable being
assigned. The source column is the data type of the value assigned to the declared 
variable.


|Declared type|Operator|Source|Description|
|---|---|---|---|
|int|:=|identifier|Allowed if data type of identifier or return type of function is int.|
|int|:=|int|Always allowed.|
|int|:=|float|Not allowed. Cannot assign a float value to an int variable.|
|float|:=|identifier|Allowed if data type of identifier or return type of function is float or int.|
|float|:=|int|Allowed. Example: int(1) becomes float(1.0) etc.|
|float|:=|float|Always allowed.|

## Expression

Binary operators.

|Left|Operator|Right|Result|
|---|---|---|---|
|int|+|int|int|
|int|-|int|int|
|int|*|int|int|
|int|/|int|int|
|int|%|int|int|
|int|<<|int|int|
|int|&#62;&#62;|int|int|
|int|&#124;|int|int|
|int|&|int|int|
|int|^|int|int|

|Left|Operator|Right|Result|
|---|---|---|---|
|int|+|float|float|
|int|-|float|float|
|int|*|float|float|
|int|/|float|float|
|int|%|float|Undefined|
|int|<<|float|Undefined|
|int|&#62;&#62;|float|Undefined|
|int|&#124;|float|Undefined|
|int|&|float|Undefined|
|int|^|float|Undefined|

|Left|Operator|Right|Result|
|---|---|---|---|
|float|+|int|float|
|float|-|int|float|
|float|*|int|float|
|float|/|int|float|
|float|%|int|Undefined|
|float|<<|int|Undefined|
|float|&#62;&#62;|int|Undefined|
|float|&#124;|int|Undefined|
|float|&|int|Undefined|
|float|^|int|Undefined|

|Left|Operator|Right|Result|
|---|---|---|---|
|float|+|float|float|
|float|-|float|float|
|float|*|float|float|
|float|/|float|float|
|float|%|float|Undefined|
|float|<<|float|Undefined|
|float|&#62;&#62;|float|Undefined|
|float|&#124;|float|Undefined|
|float|&|float|Undefined|
|float|^|float|Undefined|

Unary operators.

|Operator|Right|Result|
|---|---|---|
|~|int|int|
|-|int|int|
|~|float|Undefined|
|-|float|float|

## Relation

|Left|Operator|Right|Result|
|---|---|---|---|
|int|=|int|int|
|int|<|int|int|
|int|&#62;|int|int|

|Left|Operator|Right|Result|
|---|---|---|---|
|int|=|float|float|
|int|<|float|float|
|int|&#62;|float|float|

|Left|Operator|Right|Result|
|---|---|---|---|
|float|=|int|float|
|float|<|int|float|
|float|&#62;|int|float|

|Left|Operator|Right|Result|
|---|---|---|---|
|float|=|float|float|
|float|<|float|float|
|float|&#62;|float|float|