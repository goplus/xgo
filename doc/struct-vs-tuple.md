# Tuple vs. Struct

In the XGo programming language, tuple and struct are two distinct ways of organizing data. While both can combine multiple values together, they differ significantly in their type system, visibility rules, runtime characteristics, and other aspects. Understanding these differences is crucial for choosing the right data structure.

## Fundamental Differences in the Type System

Tuple and struct behave very differently in the type system.  Tuple types with the same element structure have identical underlying representations, meaning `(a int, b string)` and `(c int, d string)` are **identical** types. However, named tuple types are distinct. For example, `type Point (x int, y int)` and `type Coord (a int, b int)` define two different types, even though they have the same underlying structure. In this context, tuple field names are compile-time aliases, but named tuple types provide nominal typing.

In contrast, struct type checking is much stricter. Even if two structs have exactly the same field types and order, they are considered different types if their definitions differ or their field names are different. This characteristic makes struct more suitable for expressing data structures with clear semantics.

## Differences in Visibility Rules

Regarding visibility control, tuple adopts a simpler strategy: all fields in a tuple are **always public**, with no concept of lowercase letters indicating private access. This design reflects tuple's positioning as a lightweight data container—it's primarily used for temporarily combining data rather than encapsulating complex object state.

Struct, on the other hand, fully supports Go-style visibility control, using uppercase and lowercase initial letters to distinguish between public and private fields, providing necessary support for modular design and encapsulation.

## Runtime Reflection Differences

When it comes to runtime reflection, the differences become even more pronounced. After performing reflect operations on a tuple, its field names become `X_0`, `X_1`, and so on. This means that the friendly field names used at compile time **only exist during compilation** and are erased at runtime.

In contrast, struct field names are fully preserved at runtime, which enables struct to support various reflection-based functionalities such as serialization, ORM mapping, configuration parsing, and more. This is a significant advantage of struct over tuple.

## Methods and Object-Orientation

In XGo's design philosophy, **tuple does not encourage objectification**, meaning it's not recommended to add methods to tuples. This aligns with tuple's positioning as a simple data container—it should remain lightweight and simple, avoiding the burden of excessive behavioral logic.

If methods are genuinely needed for a data structure, XGo recommends using [classfile](classfile.md) to achieve more complete object-oriented features.

## Practical Application Limitations

In practical applications, these differences lead to obvious usage limitations. For example, in scenarios like **reading configuration files**, tuple cannot replace struct. Configuration parsing typically relies on reflection mechanisms to map configuration items to data structure fields, and since tuple loses field name information at runtime, it cannot support this kind of mapping.

More broadly, almost all **functionalities that depend on reflect must use struct**. Common scenarios including JSON/XML serialization, database ORM, dependency injection, and struct tag parsing all require the complete runtime type information that struct provides.

## Summary

Tuple and struct each have their appropriate use cases in XGo. Tuple is suitable as a lightweight, temporary data container for returning multiple values from functions or simple data combinations. Struct, however, is more appropriate for defining data types with clear semantics, especially in scenarios requiring encapsulation, reflection support, or method binding.

Understanding these differences helps developers choose the appropriate data structure for the right scenarios, leading to clearer and more efficient code.
