<!-- Context: ui/react-patterns | Priority: high | Version: 1.1 | Updated: 2026-02-22 -->

# React Patterns

Use functional components with hooks, and prefer composition over deeply nested prop passing. Keep components small, predictable, and explicit about loading/error states.

## Key Points

- Start with local state; lift state or use context only when sharing is required.
- Extract reusable behavior into custom hooks (`useX`) instead of duplicating effects.
- Use `useMemo` and `useCallback` only for measurable render bottlenecks.
- Keep effect dependencies correct and avoid derived-state effects when plain computation works.
- Split large screens/components with lazy loading to keep first render fast.

## Minimal Example

```tsx
function useUser(id: string) {
  const [user, setUser] = useState<User | null>(null)
  useEffect(() => { void api.getUser(id).then(setUser) }, [id])
  return user
}
```

## Practical Checklist

- Model async UI with explicit `loading`, `error`, and `ready` states.
- Prefer composition/context over prop drilling through 3+ levels.
- Co-locate component tests with behavior-critical components.
- Avoid mutation; always update with immutable state transitions.

## Common Anti-Patterns

- Overusing `useEffect` for values that can be computed during render.
- Passing unstable inline callbacks into memoized children repeatedly.
- Using array index as key for dynamic/reordered lists.
- Building "god components" with multiple unrelated responsibilities.

## References

- https://react.dev/learn
- https://react.dev/reference/react/useEffect
- https://react.dev/reference/react/useMemo

## Related

- `ui-styling-standards.md`
- `design-systems.md`
