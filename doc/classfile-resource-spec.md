# The XGo classfile resource specification

This document defines the syntax and semantics of resources in XGo classfile frameworks.

A resource is a framework-defined named entity that is visible to static analysis.

In particular, this document defines:
- how a framework declares resource kinds and type-level resource bindings
- how a framework discovers concrete resource instances through DQL over pack documents
- how a framework declares API-position scope bindings for scoped resource references
- how work classfiles may imply top-level resource identities
- how source-level resource references participate in standardized static semantics

## Terms

The following terms are used throughout this document:
- Resource kind: a framework-defined resource category such as `sprite`, `sound`, or `sprite.costume`
- Top-level kind: a resource kind with no direct parent kind
- Scoped kind: a resource kind whose instances exist within the scope of another resource kind
- Framework package: the package named by the first package path of one project group
- String-based type: an exported defined type or exported alias whose type has underlying type `string`
- Canonical resource reference type: a string-based type bound to one resource kind
- Handle-bearing type: an exported defined interface type or exported defined struct type whose values may carry one
  top-level resource identity of one resource kind
- Local resource name: the name of one resource instance within its direct parent scope, or at top level if its
  resource kind is top-level
- Scope chain: the ordered chain of ancestor resource identities from the outermost ancestor to the direct parent
- Scope-unknown reference: a scoped resource reference whose complete scope chain is not statically available
- Framework registration identity: the identity of one project group in the active classfile registry
- Resource identity: the stable logical identity of a resource instance, independent from storage path, manifest path,
  URI syntax, and runtime object identity
- Resource instance: one concrete resource available to static analysis in one classfile project under one framework
  registration
- Resource-discovery comment: a framework source directive comment that declares one concrete resource discovery query
  for one canonical resource reference type
- Resource-name-discovery comment: a framework source directive comment that declares one local resource name discovery
  query for one canonical resource reference type
- Pack document: the pack document defined by the XGo classfile specification
- Discovery origin node: one node matched by a resource-discovery comment query and associated with one discovered
  resource instance
- Callable site: one framework function declaration, one framework method declaration, or one interface method spec
  declared by one top-level handle-bearing interface type declaration
- API position: the receiver or one numbered parameter position of one callable site
- Resource-api-scope-binding comment: a framework source directive comment that declares one direct API-position scope
  binding for one scoped canonical resource reference type parameter position
- Resource-bearing work file kind: a work file kind whose work classfiles each imply one top-level resource instance
- Resource set: the set of concrete resource instances available to one implementation for one classfile project and one
  framework registration
- Resource reference: a source-level reference to a resource

The logical identity of a resource instance is the tuple:
- framework registration identity
- resource kind
- optional scope chain
- local resource name

In this specification, the resource kind in a resource identity is the instance's own kind. It does not encode any
concrete ancestor identities, even when the kind spelling uses dotted segments such as `sprite.costume.frame`.

## Notation

The syntax in this document is specified using EBNF. It uses the same EBNF conventions as the XGo classfile
specification and the Go specification.

```ebnf
ResourceComment = "//xgo:class:resource" ResourceKind .
ResourceDiscoveryComment = "//xgo:class:resource-discovery" StandardDQLQuery .
ResourceNameDiscoveryComment = "//xgo:class:resource-name-discovery" StandardDQLQuery .
ResourceAPIScopeBindingComment = "//xgo:class:resource-api-scope-binding" ScopeBindingTarget ScopeBindingSource .

ResourceKind = ResourceSegment { "." ResourceSegment } .
ResourceSegment = lower_letter { lower_letter | decimal_digit | "_" } .
ScopeBindingTarget = ParameterPosition .
ScopeBindingSource = "receiver" | ParameterPosition .
ParameterPosition = "param." decimal_digit { decimal_digit } .
lower_letter = "a" ... "z" .
```

The lexical production `decimal_digit` is as in the Go specification. The directive comment form is as in Go directive
comment conventions.

`StandardDQLQuery` is the remaining text of one resource-discovery or resource-name-discovery comment line and must be
one standard DQL query.

`ResourceComment` attaches to one immediately following top-level type spec that declares exactly one exported type name
and no type parameters.

`ParameterPosition` uses zero-based decimal indexing in ordinary parameter source order and must not use leading zeros
other than `0` itself.

In one dotted `ResourceKind`, each `.` separates one nested resource-kind segment from its direct parent prefix. A
single-segment kind is top-level. A multi-segment kind is scoped. The direct parent kind of one multi-segment kind is
the prefix that remains after removing its final `.` segment.

## Conformance

An implementation conforms to this specification if it implements the syntax and semantics defined by this specification
and satisfies every rule stated using the terms "must" and "must not".

Rules stated using "may" describe permitted behavior. Rules stated using "should" describe recommended behavior.

Capabilities such as hover, completion, diagnostics, rename, and references are optional tool capabilities. If an
implementation provides one of those capabilities for standardized resource semantics, the capability must satisfy all
applicable "must" and "must not" rules in this specification.

## Framework source metadata

Framework source metadata declares which resource kinds exist, how source code refers to them, which canonical resource
reference types carry discovery queries and optional local-name discovery queries, and how framework callable sites may
provide explicit direct scope to scoped resource positions.

### Resource comments

A resource comment belongs to the immediately following top-level type spec that declares exactly one exported type name
and no type parameters in the framework package of the containing project group.

A resource comment on any other declaration, on one grouped type declaration as a whole rather than on one contained
type spec, or in any other package, has no standardized meaning in this specification.

A declaration must not bear more than one resource comment with standardized meaning.

A resource kind is declared when a resource comment with standardized meaning names it.

If the declaration bearing a resource comment is a string-based type:
- the comment declares the canonical resource reference type of the named resource kind
- the declaration is the canonical resource reference type declaration of that kind

If the declaration bearing a resource comment is an exported defined interface type or exported defined struct type:
- the comment declares a handle-bearing type of the named resource kind
- the named resource kind must be top-level

If the declaration bearing a resource comment is neither a string-based type nor an exported defined interface type nor
an exported defined struct type, the comment has no standardized meaning in this specification.

Within one project group:
- the direct parent kind of each scoped kind must be declared in the same project group
- a resource kind must not have more than one canonical resource reference type
- a declaration that bears a resource comment with standardized meaning must belong to the framework package of the
  containing project group

A top-level resource kind may have zero, one, or more handle-bearing types.

A handle-bearing type declaration does not by itself imply any resource instance and does not by itself define any
source-level resource-binding rule.

Additional user-facing spellings may be expressed through ordinary Go aliases that reduce to the canonical resource
reference type of a kind and do not bear their own resource comments.

### Resource-discovery comments

A resource-discovery comment belongs to the immediately following declaration only if all of the following hold:
- the declaration is the canonical resource reference type declaration of one resource kind
- the declaration is in the framework package of the containing project group

A resource-discovery comment on any other declaration, in any other package, or on the same declaration as another
resource-discovery comment, has no standardized meaning in this specification.

Each canonical resource reference type declaration may bear at most one resource-discovery comment with standardized
meaning.

A resource-discovery comment declares one DQL query for discovering concrete resource instances of that resource kind.

### Resource-name-discovery comments

A resource-name-discovery comment belongs to the immediately following declaration only if all of the following hold:
- the declaration is the canonical resource reference type declaration of one resource kind
- the declaration is in the framework package of the containing project group
- the same declaration bears one resource-discovery comment with standardized meaning

A resource-name-discovery comment on any other declaration, in any other package, or on the same declaration as another
resource-name-discovery comment, has no standardized meaning in this specification.

Each canonical resource reference type declaration may bear at most one resource-name-discovery comment with
standardized meaning.

A resource-name-discovery comment declares one DQL query for discovering local resource names of that resource kind
relative to discovery origin nodes.

### Resource-api-scope-binding comments

A resource-api-scope-binding comment belongs to the immediately following callable site only if all of the following
hold:
- the callable site is one top-level function declaration, one top-level method declaration, or one method spec declared
  by one top-level handle-bearing interface type declaration
- the callable site is in the framework package of the containing project group

A resource-api-scope-binding comment on any other callable site, in any other package, or on the same callable site as
another resource-api-scope-binding comment with the same target parameter position, has no standardized meaning in this
specification.

Each callable site may bear zero or more resource-api-scope-binding comments with standardized meaning, but at most one
such comment may target one parameter position.

A set of resource-api-scope-binding comments with standardized meaning on one callable site must induce an acyclic
directed relation over API positions.

The meaning of one resource-api-scope-binding comment is independent of its source order relative to other such comments
on the same callable site.

A resource-api-scope-binding comment declares one direct scope source for its target parameter position.

## Concrete resource introduction

Concrete resource introduction in this specification occurs either through resource-discovery comments and optional
resource-name-discovery comments over pack documents or through work classfile implication.

### Discovery-based introduction

Resource-discovery comments and optional resource-name-discovery comments introduce project-derived resource instances
over pack documents.

#### Discovery execution

For a top-level resource kind, the resource-discovery comment query is evaluated on the root node of the pack document,
if any, derived from the active project group identified by the current framework registration identity.

For a scoped resource kind whose direct parent kind is `<parentKind>`, the resource-discovery comment query is evaluated
relative to each discovery origin node of each discovered direct parent resource instance of kind `<parentKind>`.

Relative evaluation means that the discovery origin node is used as the DQL query root for that evaluation.

If a direct parent resource identity is available only from work-classfile implication and not from any discovery origin
node, that implied identity alone does not create a relative discovery root.

An implementation must preserve at least one discovery origin node for each discovered resource instance.

If one discovered resource identity is obtained from more than one discovery origin node, relative child discovery is
evaluated for each such origin node. Child identities obtained from those evaluations are merged by resource identity.

If a canonical resource reference type declaration bears a resource-name-discovery comment, its query is evaluated
relative to each discovery origin node produced by the resource-discovery comment on that declaration.

Relative evaluation of a resource-name-discovery comment does not change the discovery origin node associated with the
discovered resource instance and does not create any relative discovery root for child discovery.

#### Discovery result interpretation

A successful resource-discovery comment query match contributes one discovered resource instance candidate.

For the rules below:
- a string scalar value is one pack-document string scalar value
- a node key name is the key by which one matched node appears as one member of its containing pack-document object, if
  any
- a string member named `name` is one object member named `name` whose value is a string scalar value

The candidate's local resource name is determined as follows:
- if a resource-name-discovery comment is present:
  - if its relative evaluation for that candidate does not produce exactly one matched node, the candidate is invalid
    for discovery and does not contribute a resource instance
  - if that matched node denotes a string scalar value, that string value is the local resource name
  - otherwise, if the matched node has a non-empty node key name, that key name is the local resource name
  - otherwise, if the matched node has a string member named `name`, the value of that member is the local resource name
  - otherwise, the candidate is invalid for discovery and does not contribute a resource instance
- otherwise:
  - if the discovery origin node has a non-empty node key name, that key name is the local resource name
  - otherwise, if the discovery origin node has a string member named `name`, the value of that member is the local
    resource name
  - otherwise, the candidate is invalid for discovery and does not contribute a resource instance

For a top-level kind, the discovered identity is `(resource kind, local resource name)`.

For a scoped kind, the discovered identity is `(resource kind, scope chain, local resource name)`, where the direct
parent identity is inherited from the current relative discovery execution context.

### Work classfile implication

If a handle-bearing type declaration bearing a resource comment is the registered work base class declaration of one or
more work file kinds, each such work file kind is resource-bearing.

A resource-bearing work file kind implies one top-level resource instance for each work classfile of that kind in the
analyzed classfile project.

The implied local resource name is the class file stem of the work classfile, before any class type naming normalization
or `-prefix=` application.

The implied resource identity is independent from generated Go type naming, including `-prefix=` adjustments.

This rule standardizes one classfile-native top-level resource identity. Framework-specific runtime lookup keys,
reflection binding keys, and generated Go identifiers remain framework-defined.

This rule affects only the static resource model. Compilation and runtime behavior remain unchanged.

Only work classfiles may imply resources through this rule.

A work file kind must not imply more than one top-level resource kind through this rule.

If two or more work classfiles imply the same resource identity in one analyzed classfile project:
- the identities collide
- an implementation may report a duplicate-resource diagnostic
- the colliding inputs must not be treated as distinct resource instances

Implied resources from project classfiles and scoped classfile-implied resources are outside the scope of this
specification.

### Identity merging

If more than one source contributes the same resource identity, including resource-discovery comments and work-classfile
implication:
- they refer to the same logical resource instance
- metadata may merge
- the contributing sources must not change the resource kind, scope chain, or local resource name of that identity

An implementation may preserve one origin, many origins, or provenance in a different internal form, subject to the
relative-discovery requirements above.

### Example

```go
// SpriteName identifies a sprite by name.
//
//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName = string

// SpriteCostumeName identifies a sprite costume by name.
//
//xgo:class:resource sprite.costume
//xgo:class:resource-discovery costumes.*
type SpriteCostumeName = string

// WidgetName identifies a widget by name.
//
//xgo:class:resource widget
//xgo:class:resource-discovery widgets.*
//xgo:class:resource-name-discovery id
type WidgetName = string

// SpriteImpl is a handle-bearing type of the sprite resource kind.
//
//xgo:class:resource sprite
type SpriteImpl struct{}
```

If one pack document contains `sprites.Hero`, the `sprite` query may discover `sprite(Hero)` and preserve `sprites.Hero`
as one discovery origin node of that resource. The `sprite.costume` query is then evaluated relative to that origin
node, so it may discover costumes of `Hero` without embedding `Hero` into the query text.

If one `widget` origin node stores its local resource name in a child member `id` rather than in its node key or a child
member `name`, the `resource-name-discovery` query `id` is evaluated relative to that origin node and yields the local
resource name.

If the containing project group declares `class .spx SpriteImpl`, the resource comment on `SpriteImpl` makes `.spx`
resource-bearing, and each `.spx` work classfile implies one top-level `sprite` resource instance whose local resource
name is the class file stem.

## Static semantics

### Resource references

A source position participates in standardized resource semantics only if ordinary Go typing determines one canonical
resource reference type declaration of one resource kind for that position.

For this purpose, a canonical resource reference type declaration is determined for a source expression in one of the
following ways:
- the expression's static type is that canonical resource reference type declaration, or can be reduced to it by
  following Go alias declarations only
- the surrounding typed position requires that canonical resource reference type, and the expression is assignable to it
  under ordinary Go typing rules

A distinct defined type does not participate in standardized resource semantics solely because both it and a canonical
resource reference type have underlying type `string`.

A Go alias declaration denotes the same resource kind as that canonical resource reference type if its denoted type can
be reduced, recursively through Go alias declarations only, to the same canonical resource reference type declaration.

An expression is canonically resource-typed if one canonical resource reference type declaration is determined for it in
that way.

For one canonically resource-typed expression:
- if the expression is a string literal or a statically evaluable string constant, it is a resource reference candidate
- if the expression cannot be statically evaluated, it remains resource-typed but is not a resolvable resource reference
- for a top-level kind, the lookup key is `(resource kind, local resource name)`
- for a scoped kind, the lookup key is `(resource kind, scope chain, local resource name)`
- if a scoped kind does not have a statically available complete scope chain, the reference is one `scope-unknown`
  reference

### API-position scope bindings

In this subsection, the callable site is the function declaration, method declaration, or interface method spec to which
one resource-api-scope-binding comment with standardized meaning belongs.

One resource-api-scope-binding comment has standardized meaning only if all of the following hold:
- the target parameter position exists on the callable site
- the target parameter position is not the variadic parameter position of the callable site
- the target parameter type determines one canonical resource reference type declaration of one scoped kind
- the source API position exists on the callable site
- if the source API position is one parameter position, it is not the variadic parameter position of the callable site
- the source API position determines either:
  - one canonical resource reference type declaration of the direct parent kind of the target kind, or
  - one handle-bearing type of that direct parent kind

At one call that resolves to that callable site, the source API position is interpreted as follows:
- if the source API position is `receiver`, the source expression is the call's receiver expression, whether explicit or
  implicit
- if the source API position is one parameter position, the source expression is the corresponding argument expression

One resource-api-scope-binding comment contributes one explicit direct parent identity to its target argument position
only if the source expression yields one exact resource identity of the target kind's direct parent kind under the
resource semantics otherwise available at that source position.

For this purpose, a source position may itself use explicit scope contributed by other resource-api-scope-binding
comments on the same callable site, subject to the acyclicity rule above.

If the target kind is multiply scoped, a resource-api-scope-binding comment contributes at most the direct parent
identity. Any additional ancestor levels come from that direct parent identity's own scope chain, if available.

If a target position receives one explicit direct parent identity from a resource-api-scope-binding comment, that
identity is explicit scope for that target position.

Example:

```go
//xgo:class:resource-api-scope-binding param.0 receiver
func (p *SpriteImpl) SetCostume__0(costume SpriteCostumeName)

//xgo:class:resource-api-scope-binding param.0 receiver
//xgo:class:resource-api-scope-binding param.1 param.0
func (p *SpriteImpl) SetCostumeAndFrame(costume SpriteCostumeName, frame SpriteCostumeFrameName)
```

If one call to `SetCostume__0` yields one exact `sprite` identity at its receiver position, the bound
`SpriteCostumeName` argument position may use that identity as explicit scope.

If one call to `SetCostumeAndFrame` yields one exact `sprite` identity at its receiver position, `param.0` may use that
identity as explicit scope. If `param.0` then yields one exact `sprite.costume` identity, the bound
`SpriteCostumeFrameName` argument position may use that identity as explicit scope.

### Scoped owner inference

If all of the following hold:
- the current source position is inside a work classfile
- the registered work base class declaration of that work file kind bears a resource comment for one top-level kind
  `<parentKind>`
- the referenced scoped kind has direct parent kind `<parentKind>`
- no explicit scope is otherwise statically available

then the statically available direct parent identity is the implied top-level resource of the containing work classfile.

Example:
- if the declaration of `SpriteImpl` bears `//xgo:class:resource sprite`
- and `class .spx SpriteImpl` is active
- and `SpriteCostumeName` bears `//xgo:class:resource sprite.costume`
- then a scoped `SpriteCostumeName` reference inside `Hero.spx` may use scope `(sprite, Hero)`

If a resource-api-scope-binding comment contributes explicit scope to one target position, that explicit scope takes
precedence over the narrow rule above for that target position.

Receiver-bound owner inference, project auto-binding owner inference, and other framework-specific context rules remain
framework-specific.

If the narrow rule above does not apply, the scoped reference remains `scope-unknown`.

The narrow rule above contributes at most one statically available parent identity. It does not by itself infer any
additional ancestor levels for multiply scoped kinds.

If a framework uses plain `string` parameters or resource-like values that do not reduce to canonical resource reference
types, those positions are outside the standardized resource semantics of this specification.

### Diagnostics

A conforming implementation:
- may emit a "resource not found" diagnostic only when its framework-specific discovery semantics are exact for the
  analyzed project state
- must not emit a "resource not found" diagnostic for dynamic expressions
- must not emit a "resource not found" diagnostic for `scope-unknown` references
- may emit a diagnostic for an empty resource name

## Tool semantics

### Hover

When a source position resolves to a resource reference whose identity is present in the implementation's resource set,
an implementation that provides hover must return information about the corresponding resource instance.

### Completion

When the target type at an input position resolves to a resource kind, an implementation that provides completion may
base resource completion candidates on the implementation's resource set.

For scoped kinds:
- if the scope chain is known, completion must be filtered to that scope chain
- if the scope chain is unknown, an implementation may suppress completion or provide degraded completion annotated with
  scope information

### Rename and references

If an implementation supports rename or reference lookup for resources, resource identity must be based on
`(framework registration identity, resource kind, optional scope chain, local resource name)` rather than raw string
text.

## Excluded semantics

This specification intentionally does not standardize:
- framework-specific API-position scope rules beyond explicit resource-api-scope-binding comments
- API-position kind binding rules for positions that do not already depend on canonical resource reference types
- implied resources from project classfiles
- scoped classfile-implied resources beyond the work-classfile implication rule defined by this specification
- how runtime parameters such as `run(...)` are mapped back into static analysis
- the concrete encoding of preview URIs or editor-specific payloads
- one standardized completeness or exactness model for resource discovery
