# XGo Project AI Assistant Guide

## Workflow

### Adding New Syntax Features

When implementing new language syntax, follow this three-phase approach:

**IMPORTANT**: Each phase must be implemented in a separate pull request. Do NOT mix phases in a single PR. This separation ensures:
- Clear review focus (grammar vs semantics vs documentation)
- Easier rollback if issues are found
- Better git history and maintainability
- Allows grammar to be reviewed independently from implementation details

#### Phase 1: Grammar Definition (First Pull Request)
**Scope**: AST, parser, and printer modifications ONLY
- **AST**: Define new node types in `ast/` directory (if needed - often existing nodes can be reused)
- **Parser**: Implement parsing rules in `parser/` directory to recognize the new syntax
- **Printer**: Add formatting support for new syntax (inverse of parsing) in `printer/` directory
- **Testing**: Add test cases in `parser/_testdata/` for new syntax
  - **Note**: Printer shares test cases with parser - do NOT create separate test files in `printer/_testdata/`
- **What NOT to include**: Do NOT add any code generation or semantic logic in `cl/` package - that belongs in Phase 2

#### Phase 2: Semantic Implementation (Second Pull Request)
**Scope**: Code generation via `cl` package ONLY
- **Code Generation**: Implement semantics using `github.com/goplus/gogen` package
- **Type Safety**: Leverage gogen's type information maintenance for semantic correctness
- **Testing**: Add comprehensive test cases in `cl/_testxgo/` covering various usage scenarios
- **Prerequisite**: Phase 1 PR must be merged before starting Phase 2

When submitting a new PR to the `gogen` repository for a change, also submit a PR to the `xgo` repository that updates the `gogen` module to the latest code from the corresponding `gogen` branch. The xgo PR's go.mod/go.sum must pin the gogen dependency to the feature branch's commit (not to a released version), so the two PRs represent a coherent, reviewable change set.

#### Phase 3: Documentation (Third Pull Request)
**Scope**: User-facing documentation updates ONLY
- **Quick Start Guide**: Add feature documentation to `doc/docs.md` with practical examples
- **Table of Contents**: Update TOC in quick start to include new feature section
- **Language Specification**: Update specification documents (see Language Specification Structure below)
- **Examples**: Provide clear, runnable code examples demonstrating the feature
- **Prerequisite**: Phase 2 PR must be merged before starting Phase 3

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
- **Phase 2**: Comprehensive test coverage for semantic implementation in `cl/_testxgo/` covering:
  - Common usage scenarios
  - Edge cases and error conditions
  - Integration with existing type system
- **Phase 3**: Documentation validation
  - Ensure all code examples in documentation are runnable and correct
  - Verify documentation accurately reflects implemented behavior
  - Check that TOC links work correctly
- **Coverage rule**: Unit test coverage **must not decrease** across any PR. Run `go test ./... -cover` before and after your changes and compare. If coverage drops for any package, add tests to compensate before submitting.
