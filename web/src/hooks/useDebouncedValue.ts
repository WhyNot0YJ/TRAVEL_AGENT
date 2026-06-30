import { useEffect, useState } from "react";

// useDebouncedValue returns the value after `delay` ms of no changes.
// Useful for search-as-you-type without spamming the API.
export function useDebouncedValue<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(handle);
  }, [value, delay]);
  return debounced;
}
