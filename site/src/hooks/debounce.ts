import { useCallback, useEffect, useRef, useState } from "react";

type useDebouncedFunctionReturn<Args extends unknown[]> = Readonly<{
  debounced: (...args: Args) => void;

  // Mainly here to make interfacing with useEffect cleanup functions easier
  cancelDebounce: () => void;
}>;

/**
 * Creates a debounce function that is resilient to React re-renders, as well as
 * a function for canceling a pending debounce.
 *
 * The returned-out functions will maintain the same memory references, but the
 * debounce function will always "see" the most recent versions of the arguments
 * passed into the hook, and use them accordingly.
 *
 * If the debounce time changes while a callback has been queued to fire, the
 * callback will be canceled completely. You will need to restart the debounce
 * process by calling debounced again.
 */
export function useDebouncedFunction<
  // Parameterizing on the args instead of the whole callback function type to
  // avoid type contra-variance issues; want to avoid need for type assertions
  Args extends unknown[] = unknown[],
>(
  callback: (...args: Args) => void | Promise<void>,
  debounceTimeMs: number,
): useDebouncedFunctionReturn<Args> {
  const timeoutIdRef = useRef<number | null>(null);
  const cancelDebounce = useCallback(() => {
    // Clearing timeout because, even though hot-swapping the timeout value
    // while keeping any active debounced functions running was possible, it
    // seemed like it'd be ripe for bugs. Can redesign the logic if that ends up
    // becoming an actual need down the line.
    if (timeoutIdRef.current !== null) {
      window.clearTimeout(timeoutIdRef.current);
    }

    timeoutIdRef.current = null;
  }, []);

  const debounceTimeRef = useRef(debounceTimeMs);
  useEffect(() => {
    cancelDebounce();
    debounceTimeRef.current = debounceTimeMs;
  }, [cancelDebounce, debounceTimeMs]);

  const callbackRef = useRef(callback);
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  // Returned-out function will always be synchronous, even if the callback arg
  // is async. Seemed dicey to try awaiting a genericized operation that can and
  // will likely be canceled repeatedly
  const debounced = useCallback(
    (...args: Args): void => {
      cancelDebounce();

      timeoutIdRef.current = window.setTimeout(
        () => void callbackRef.current(...args),
        debounceTimeRef.current,
      );
    },
    [cancelDebounce],
  );

  return { debounced, cancelDebounce } as const;
}

export function useDebouncedValue<T = unknown>(
  value: T,
  debounceTimeMs: number,
): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      setDebouncedValue(value);
    }, debounceTimeMs);

    return () => window.clearTimeout(timeoutId);
  }, [value, debounceTimeMs]);

  return debouncedValue;
}
