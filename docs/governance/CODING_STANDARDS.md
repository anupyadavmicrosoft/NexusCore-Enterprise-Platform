# NexusCore Coding Standards

## 1. Scope & Objective

This document defines the mandatory programming style, safety guidelines, and language-specific conventions for the NexusCore engineering codebase. 

Compliance with this standard is enforced through automated linters (`golangci-lint` for Go, `eslint` for TypeScript/React) and manual peer reviews. No code that violates these standards may be merged into the mainline branch.

---

## 2. General Style Guidelines

Across all languages, projects, and directories:

### 2.1 Naming Conventions
*   **Clarity over Conciseness**: Variable names must explicitly state their purpose. Avoid single-character variable names except for index variables in short loops (`i`, `j`).
    *   *Bad*: `t := time.Now()`, `u := getUser()`
    *   *Good*: `startTime := time.Now()`, `userRecord := getUser()`
*   **Acronyms**: Keep acronyms consistent in capitalization. Use `HTTPResponse` instead of `HttpResponse`, and `userID` instead of `userId`.
*   **Boolean Variables**: Prefix boolean variables with helper verbs like `is`, `has`, `should`, or `can`.
    *   *Example*: `isAuthorized`, `hasReplica`, `shouldRetry`.

### 2.2 Formatting & Linting
*   **Automated Formatting**: You must run the language formatter before committing code (`go fmt` for Go, `prettier` or `eslint --fix` for TypeScript/React).
*   **Max Line Length**: Limit lines to a maximum of **120 characters** to ensure readability on modern monitors without horizontal scrolling.
*   **Indentation**:
    *   **Go**: Standard tabs (enforced by `gofmt`).
    *   **TypeScript/React**: 2 spaces (enforced by `.prettierrc`).

---

## 3. Go (Golang) Specific Standards

### 3.1 Error Handling
Error handling is Go's explicit control mechanism. We treat errors as values that must be handled immediately:
*   **No Ignored Errors**: Never ignore returned errors. Do not assign errors to the blank identifier (`_`).
    *   *Exception*: Logging or writing to a standard writer where failure has no operational effect (even then, log the failure if appropriate).
*   **Error Wrapping**: Always wrap lower-level errors with operational context using `%w` in `fmt.Errorf`. This preserves the original error type for upstream assertions using `errors.Is` or `errors.As`.
    ```go
    // Good
    user, err := r.db.GetUser(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve user profile for id %s: %w", id, err)
    }
    ```
*   **Error Handling Locations**: Keep the happy path left-aligned. Handle errors in nested `if` blocks and return early.
    ```go
    // Bad
    if err == nil {
        // ... long block of code
    } else {
        return err
    }

    // Good
    if err != nil {
        return err
    }
    // ... happy path remains left-aligned
    ```

### 3.2 Concurrency Guidelines
Go makes concurrency easy, but correct concurrency is hard. Follow these safety parameters:
*   **Goroutine Lifetime**: Never spawn a goroutine without knowing how and when it will terminate. Leaking goroutines will exhaust system memory.
*   **Channels vs Mutexes**: Prefer channels for communication of state/ownership, and mutexes (`sync.Mutex`, `sync.RWMutex`) for mutual exclusion on raw memory structures.
*   **Unlock with defer**: When lock-guarding a block, call `mu.Lock()` and immediately follow with `defer mu.Unlock()`. Do not manually unlock unless performance profiling requires microscopic optimization.
    ```go
    func (s *SafeStore) Set(key string, val interface{}) {
        s.mu.Lock()
        defer s.mu.Unlock()
        s.data[key] = val
    }
    ```
*   **Race Detector**: All local, CI, and staging tests must run with the `-race` flag enabled (`go test -race ./...`). Any data race warning is considered a blocking build failure.

### 3.3 Slice & Map Allocation
*   **Pre-allocate Slices**: When the size of a slice is known beforehand, allocate it with `make([]T, 0, capacity)` to avoid performance hits from repeated memory reallocation.
*   **Avoid Map Leaks**: Go maps do not shrink in memory after items are deleted. For long-running processes dealing with transient map data, periodically recreate the map or use a custom cache store.

---

## 4. TypeScript & React Specific Standards

### 4.1 Strict Typing
*   **No `any`**: The use of `any` is strictly prohibited. Use `unknown` if the type is genuinely dynamic, and write runtime assertions or type guards.
*   **Type Declarations**: Define complex types or interfaces in a central `/src/types.ts` file if shared, or locally if isolated to a component.
*   **Enums**: Use standard TypeScript `enum` declarations instead of `const enum` to prevent build-time inlining issues.

### 4.2 React Component Standards
*   **Functional Components**: Always write React components as functional components using modern hooks. Class components are prohibited unless wrapping legacy libraries that require them.
*   **UseEffect Dependency Safety**:
    *   Never update state directly inside the component render body (causes infinite loops).
    *   Do not include mutable arrays, objects, or functions directly in the `useEffect` dependency array unless they are memoized using `useMemo` or `useCallback`.
    *   Prefer using primitive variables (`string`, `number`, `boolean`) in dependency arrays to prevent unnecessary triggers.
*   **Unique HTML IDs**: Ensure all interactive or structurally critical HTML components (cards, buttons, inputs) feature a unique, semantic `id` attribute. This is required for end-to-end automated testing and element targeting.
    ```tsx
    // Good
    <button 
      id="submit-payment-btn" 
      onClick={handleSubmit} 
      className="px-4 py-2 bg-blue-600 rounded"
    >
      Submit Payment
    </button>
    ```

### 4.3 Icon Library Consistency
*   All icons utilized in the user interface **MUST** be imported directly from the `lucide-react` library.
*   Do not write custom SVG markup for icons.
