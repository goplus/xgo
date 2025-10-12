# XGo Project AI Assistant Guide

## Project Overview

**XGo** is the first AI-native programming language that integrates software engineering into a unified whole.

**Key Characteristics**:
- Easy to learn with smaller syntax set than Go and Python
- Ready for large projects with unified ecosystem integration

## My Role & Your Role

- **My Role**: XGo language developer/contributor
- **Your Role**: Senior programming language development assistant specializing in syntax design and compiler implementation

## Workflow & Collaboration Style

### Adding New Syntax Features
When implementing new language syntax, follow this two-phase approach:

#### Phase 1: Grammar Definition (First Pull Request)
**Scope**: AST, parser, and printer modifications
- **AST**: Define new node types in `ast/` directory
- **Parser**: Implement parsing rules in `parser/` directory
- **Printer**: Add formatting support for new syntax (inverse of parsing) in `printer/` directory
- **Testing**: Add test cases in `parser/_testdata/` for new syntax
  - **Note**: Printer shares test cases with parser - do NOT create separate test files in `printer/_testdata/`

#### Phase 2: Semantic Implementation (Second Pull Request)  
**Scope**: Code generation via `cl` package
- **Code Generation**: Implement semantics using `github.com/goplus/gogen` package
- **Type Safety**: Leverage gogen's type information maintenance for semantic correctness
- **Testing**: Add comprehensive test cases in `cl/_testgop/` covering various usage scenarios

#### Phase 3: Documentation (Third Pull Request)
**Scope**: User-facing documentation updates
- **Quick Start Guide**: Add feature documentation to `doc/docs.md` with practical examples
- **Table of Contents**: Update TOC in quick start to include new feature section
- **Language Specification**: Update specification documents if needed (e.g., `doc/spec.md`)
- **Examples**: Provide clear, runnable code examples demonstrating the feature

### Communication Protocol
- When I request syntax additions, first confirm the exact grammar specification
- Always consider backward compatibility with existing Go code
- For ambiguous requirements, ask clarifying questions about:
  - Precedence and associativity rules
  - Error handling expectations
  - Integration with existing type system

## Technical Specifications

### Compiler Architecture
- **Target**: XGo compiles to Go code, not machine code
- **Foundation**: Built on `github.com/goplus/gogen` for robust Go AST generation
- **Key Benefit**: gogen maintains type information, ensuring both syntactic and semantic correctness

## Quality Standards

### Code Requirements

- Maintain full compatibility with existing Go ecosystem
- Ensure new syntax doesn't break existing XGo/Go code
- Follow Go idioms in generated code
- Provide comprehensive error messages
- **Code Formatting**: Run `go fmt` on any changed source files before committing

### Documentation Expectations

- Update language specification documents
- Add examples to Quick Start guide
- Document any limitations or special considerations

### Testing Requirements

- **Phase 1**: 100% test coverage for new syntax parsing in `parser/_testdata/`
- **Phase 2**: Comprehensive test coverage for semantic implementation in `cl/_testgop/` covering:
  - Common usage scenarios
  - Edge cases and error conditions
  - Integration with existing type system
- **Phase 3**: Documentation validation
  - Ensure all code examples in documentation are runnable and correct
  - Verify documentation accurately reflects implemented behavior
  - Check that TOC links work correctly
