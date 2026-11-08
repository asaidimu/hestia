/**
 * Helpers for turning callback-based SSE transports into async iterables.
 *
 * The stream stores (audit, logs, ...) used to build this queue/iterator dance
 * by hand around the browser `EventSource` global. That tied them to the DOM,
 * bypassed the transport's API-key header, and could not be cancelled
 * deterministically. `createAsyncStream` centralises the queueing so a store
 * only has to forward `openStream` callbacks into `push`/`end`.
 */

export type AsyncStreamStatus = "active" | "cancelled" | "completed"

export interface AsyncStreamController<T> {
  /** The async iterable of queued items. */
  stream(): AsyncIterable<T>
  /** Queue an item. Ignored once the stream has ended or been cancelled. */
  push(item: T): void
  /** Mark the stream complete (e.g. the server closed the connection). */
  end(): void
  /** Mark the stream cancelled (e.g. the caller aborted it). */
  cancel(): void
  /** Current lifecycle state. */
  status(): AsyncStreamStatus
}

/**
 * Create an async stream backed by an unbounded in-memory queue.
 *
 * Consumers pull with `for await`, producers call `push`, and either side can
 * stop the stream with `end`/`cancel`. `push` is safe to call before iteration
 * starts — items are buffered until the consumer reads them.
 */
export function createAsyncStream<T>(): AsyncStreamController<T> {
  const pending: T[] = []
  let status: AsyncStreamStatus = "active"
  let resolveNext: (() => void) | null = null

  const wake = () => {
    if (!resolveNext) return
    const resolve = resolveNext
    resolveNext = null
    resolve()
  }

  const push = (item: T) => {
    if (status !== "active") return
    pending.push(item)
    wake()
  }

  const end = () => {
    if (status === "active") status = "completed"
    wake()
  }

  const cancel = () => {
    if (status === "active") status = "cancelled"
    wake()
  }

  async function* iterate(): AsyncGenerator<T> {
    while (status === "active") {
      if (pending.length > 0) {
        yield pending.shift()!
        continue
      }
      await new Promise<void>((resolve) => {
        resolveNext = resolve
        // Close the race between the empty check above and registering the
        // resolver: a push/end/cancel may have landed in between.
        if (pending.length > 0 || status !== "active") {
          resolveNext = null
          resolve()
        }
      })
    }
    // Deliver items that were queued before the stream ended.
    while (pending.length > 0) {
      yield pending.shift()!
    }
  }

  return {
    stream: () => iterate(),
    push,
    end,
    cancel,
    status: () => status,
  }
}
