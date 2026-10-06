# PineGo Engine Specification

## Goal

PineGo is a Go-native Pine Script v6 execution engine.

Primary goal:

> Execute Pine Script with TradingView-compatible semantics.

PineTS is used as a reference implementation for behavior, edge cases,
architecture ideas, and regression scenarios.

PineGo does not copy PineTS implementation code.

---

## Scope

PineGo Engine includes:

1. Lexer
2. Parser
3. AST
4. Semantic Analyzer
5. Intermediate Representation
6. Runtime
7. Series / History Engine
8. Stateful Execution
9. Built-in Namespaces
10. Request / Timeframe Engine
11. Strategy Execution Model
12. Compatibility Test System

Frontend/UI is out of scope.

---

## Execution Pipeline

Pine Source
    ↓
Lexer
    ↓
Tokens
    ↓
Parser
    ↓
AST
    ↓
Semantic Analyzer
    ↓
PineGo IR
    ↓
Runtime
    ↓
Series / State
    ↓
Built-ins
    ↓
Execution Result

---

## 1. Lexer

The lexer must support:

- Numbers
- Decimal numbers
- Scientific notation
- Strings
- Multiline strings
- Boolean literals
- `na`
- Color literals
- Identifiers
- Keywords
- Contextual keywords
- Operators
- Punctuation
- Comments
- Newlines
- Indentation
- Dedentation
- Line wrapping
- Grouped expressions
- Tabs
- Spaces
- Blank lines
- CRLF

---

## 2. Parser

The parser must support:

### Declarations

- Variables
- `var`
- `varip`
- Typed variables
- Tuple declarations
- Functions
- Methods
- Types / UDT
- Enums
- Imports
- Exports

### Statements

- Expressions
- `if`
- `else`
- `for`
- `for ... in`
- `while`
- `break`
- `continue`
- `return`
- `switch`

### Expressions

- Literals
- Identifiers
- Unary operators
- Binary operators
- Assignment
- Compound assignment
- Conditional `?:`
- Function calls
- Method calls
- Member access
- History access
- Array literals
- Tuple expressions
- Nested expressions

Operator precedence and associativity must be explicitly defined.

---

## 3. Semantic Analyzer

The semantic layer handles:

- Symbol table
- Scope tree
- Variable declarations
- Variable references
- Shadowing
- Function definitions
- Method definitions
- Type information
- Series information
- `var` persistence
- `varip` persistence
- Tuple binding
- Contextual keywords
- Reserved names
- Function signatures
- Built-in resolution

---

## 4. PineGo IR

PineGo must not depend on JavaScript generation.

Example:

Pine:

    x = ta.sma(close, 20)

IR:

    Assign
      name: x
      value:
        Call
          namespace: ta
          function: sma
          args:
            close
            20

IR must be executable directly by the Go runtime.

---

## 5. Runtime

The runtime executes the script bar-by-bar.

For every bar:

1. Update market data
2. Update execution context
3. Execute script
4. Update variable state
5. Update series history
6. Execute stateful built-ins
7. Record outputs

Runtime must support:

- Historical execution
- Current bar execution
- Conditional execution
- Persistent state
- Stateful calculations
- Function-local state
- Call-site state isolation
- Lazy boolean evaluation
- Realtime execution semantics

---

## 6. Series / History

Series is a core PineGo abstraction.

Required behavior:

    close[0] → current bar
    close[1] → previous bar
    close[2] → two bars ago

Internal storage may be chronological:

    oldest → newest

History access must be reverse indexed.

Must support:

- Dynamic history offsets
- Out-of-range history
- Before-first-bar behavior
- `na`
- Boolean v6 behavior
- Scalar / series interaction
- Series offsets
- Persistent series

---

## 7. Stateful Built-ins

Each stateful function call must have isolated state.

Example:

    ta.sma(close, 20)
    ta.sma(close, 50)

must not share internal state.

The runtime must support:

- Call-site identity
- Incremental calculation
- Dynamic lengths
- Optional parameters
- Tentative/current-bar state
- Committed historical state

---

## 8. Built-in Namespaces

### Core

- `ta`
- `math`
- `array`
- `matrix`
- `map`

### Market / Time

- `request`
- `time`
- `timeframe`
- `ticker`
- `syminfo`

### Runtime / Strategy

- `barstate`
- `input`
- `strategy`

### Utilities

- `str`
- `color`

### Drawing / Output

- `plot`
- `hline`
- `fill`
- `label`
- `line`
- `linefill`
- `box`
- `table`

---

## 9. Request / Timeframe

Architecture must support:

- `request.security`
- `request.security_lower_tf`
- Higher timeframe requests
- Lower timeframe requests
- Lookahead
- Gaps
- Dynamic timeframes
- Minute timeframes
- Day timeframes
- Week timeframes
- Month timeframes
- Multi-day
- Multi-week
- Multi-month
- Calendar alignment

Timeframe parsing must be generic, not a fixed hardcoded list.

---

## 10. Strategy Engine

Strategy support is part of the engine.

Architecture must allow:

- `strategy()`
- Entries
- Exits
- Position state
- Orders
- Open trades
- Closed trades
- Position size
- Pyramiding
- Commission
- Slippage
- Order timing
- Recalculation behavior

---

## 11. Compatibility Testing

Tests are organized by behavior:

tests/
    lexer/
    parser/
    semantic/
    runtime/
    series/
    namespaces/
    compatibility/
    fixtures/

Every compatibility case should define:

- Pine source
- Expected behavior
- Expected values
- Regression test

TradingView is the behavioral reference.

PineTS is a secondary implementation reference.

Never define expected behavior solely from PineTS output.

---

## 12. Design Rules

### Rule 1
Runtime must not depend on frontend/UI code.

### Rule 2
Parser must not contain runtime logic.

### Rule 3
Lexer must not implement semantic validation.

### Rule 4
Semantic analysis must happen before execution.

### Rule 5
IR must be independent from UI.

### Rule 6
Series semantics are a first-class runtime concept.

### Rule 7
Stateful function calls must be isolated.

### Rule 8
Compatibility tests are part of the engine, not an afterthought.

### Rule 9
TradingView behavior is the final compatibility target.

### Rule 10
PineTS code is studied, not copied.

---

## Initial Milestones

### M0
Repository and architecture

### M1
Lexer

### M2
Parser + AST

### M3
Semantic Analyzer

### M4
IR

### M5
Runtime

### M6
Series / History

### M7
Core Built-ins

### M8
Request / Timeframe

### M9
Strategy Engine

### M10
Compatibility Suite

### M11
Performance Optimization

### M12
Vela Integration

---

## Definition of Engine v0.1

PineGo v0.1 is complete when it can execute a meaningful subset of
Pine Script end-to-end:

    Pine Source
        ↓
    Lexer
        ↓
    Parser
        ↓
    AST
        ↓
    Semantic Analysis
        ↓
    IR
        ↓
    Runtime
        ↓
    Series / TA
        ↓
    Result

without requiring any frontend.