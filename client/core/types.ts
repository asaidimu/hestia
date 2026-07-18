import type { QueryFilter, SortConfiguration } from "@asaidimu/query";

// Internal metadata managed by Anansi
interface DocumentMetadata {
  checksum: string; // Integrity hash of the document
  created: string; // Timestamp (often as a numeric string / nanoseconds)
  updated: string; // Timestamp (often as a numeric string / nanoseconds)
  version: number; // Optimistic locking version (increments on each write)
}

// The base envelope applied to every single document in the system
interface BaseDocument {
  _id_: string;
  _metadata_: DocumentMetadata;
}

// Generic Document type: base envelope + your custom properties T
export type Document<T extends Record<string, any>> = BaseDocument & T;

/**
 * Provides comprehensive pagination information for a collection of records.
 */
export type PaginationInfo = {
  /** The current page number (1-based). */
  number: number;

  /** The maximum number of items requested per page. */
  size: number;

  /** The number of items in the current page */
  count: number;

  /** The total count of all items across all pages. */
  total: number;

  /** The total number of available pages. */
  pages: number;
};

/**
 * Represents a single paginated response containing a list of records and pagination details.
 * @template T The type of the records in the page.
 */
export interface Page<T extends Record<string, any>> {
  /** The array of records for the current page. */
  data: Document<T>[];

  /** Indicates if the data query is currently loading. */
  loading: boolean;

  /** An error object if the query failed, otherwise `undefined`. */
  error?: any | undefined;

  /** Pagination metadata providing details about the current page and total collection. */
  page: PaginationInfo;
}

export interface PagerRefreshOptions {
  sort?: SortConfiguration | SortConfiguration[];
  filter?: QueryFilter;
  delay?: number;
  size?: number;
  page?: number;
}
/**
 * Represents the complete state and control mechanisms for consuming paginated data.
 * @template T The type of the records in the page (`TableRowData` or an extension).
 */
export interface PagedData<T extends Record<string, any>> {
  id: () => string

  /** The current paginated data, or `undefined` if not found or still loading. */
  page: () => Page<T>;

  /** Function to fetch a specific page of data.
   * @param page The 1-based page number to fetch.
   */
  navigate: (page: number) => Promise<void>;

  sort: (sort: SortConfiguration<T> | SortConfiguration<T>[]) => Promise<void>;
  filter: (filter?: QueryFilter<T>) => Promise<void>;

  /**
   * Function to change the page size of the data
   */
  resize: (size: number, page: number) => Promise<void>;

  /**
   * Function to force a refresh of the current page data, optionally with a delay.
   * @param delay Optional delay in milliseconds before the refresh operation starts.
   */
  refresh: (opts?: PagerRefreshOptions) => Promise<void>;

  /**
   * Subscribes to page changes.
   * The callback is invoked immediately with the current page, and then on every
   * subsequent change (navigation, resize, SSE patch, refresh).
   *
   * @param listener A function that receives the current Page<T>.
   * @returns An unsubscribe function.
   */
  subscribe: (listener: (page: Page<T>) => void) => () => void;

  /**
   * Invalidates the pagination controller, cleaning up any cached resources
   * (e.g., store subscriptions, in‑flight requests) and removing internal state.
   * After calling `invalidate()`, the controller should no longer be used.
   *
   * This is useful when the component using the pager is unmounted or when
   * you need to reset the pagination state completely.
   */
  invalidate: () => void;
}

/**
 * Represents an event emitted by the store, typically for notifications or state changes.
 */
export interface StoreEvent {
  /** The scope of the event, indicating its type and context. Custom scopes are allowed. */
  scope: string;
  /** Optional payload carrying data related to the event. */
  payload?: any;
}

/**
 * Defines the core interface for interacting with a remote data store.
 * @template T The type of the records managed by the store, extending Record.
 * @template TFindOptions Options for the find operation.
 * @template TReadOptions Options for the read operation.
 * @template TListOptions Options for the list operation.
 * @template TPageOptions Options for the paging operation.
 * @template TDeleteOptions Options for the delete operation.
 * @template TUpdateOptions Options for the update operation.
 * @template TCreateOptions Options for the create operation.
 * @template TUploadOptions Options for the upload operation.
 * @template TStreamOptions Options for the stream operation.
 */
export interface DocumentStore<
  T extends Record<string, any>,
  TFindOptions = Record<string, unknown>,
  TReadOptions = Record<string, unknown>,
  TListOptions = Record<string, unknown>,
  TPageOptions = Record<string, unknown>,
  TDeleteOptions = Record<string, unknown>,
  TUpdateOptions = Record<string, unknown>,
  TCreateOptions = Record<string, unknown>,
  TUploadOptions = Record<string, unknown>,
  TStreamOptions = Record<string, unknown>,
> {
  /**
   * Finds records based on provided options, returning a paginated result.
   * @param options The options for the find operation.
   * @returns A promise that resolves to a Page of records.
   */
  find: (options: TFindOptions) => Promise<Page<T>>;

  /**
   * Reads a single record by its identifier or other read options.
   * @param options The options for the read operation, typically including an ID.
   * @returns A promise that resolves to the record or undefined if not found.
   */
  read: (options: TReadOptions) => Promise<Document<T> | undefined>;

  /**
   * Lists records based on provided options, returning a paginated result.
   * @param options The options for the list operation.
   * @returns A promise that resolves to a Page of records.
   */
  list: (options: TListOptions) => Promise<Page<T>>;

  /**
   * Deletes a record based on provided options, typically including an ID.
   * @param options The options for the delete operation.
   * @returns A promise that resolves when the deletion is complete.
   */
  delete: (options: TDeleteOptions) => Promise<void>;

  /**
   * Updates an existing record.
   * @param props An object containing the ID of the record to update, the partial data, and optional update options.
   * @returns A promise that resolves to the updated record or undefined if not found.
   */
  update: (props: {
    data: Partial<T>;
    options?: TUpdateOptions;
  }) => Promise<Document<T> | undefined>;

  /**
   * Creates a new record.
   * @param props An object containing the data for the new record and optional create options.
   * @returns A promise that resolves to the newly created record or undefined if creation failed.
   */
  create: (props: {
    data: Partial<T>;
    options?: TCreateOptions;
  }) => Promise<Document<T> | undefined>;

  /**
   * Uploads a file associated with a record. Optionally updates the record with upload details.
   * @param props An object containing the file to upload and optional upload options.
   * @returns A promise that resolves to the updated record after upload or undefined if upload failed.
   */
  upload: (props: {
    file: File;
    options?: TUploadOptions;
  }) => Promise<Document<T> | undefined>;

  /**
   * Subscribes to store events for a given scope.
   * @param scope The event scope to subscribe to (e.g., 'todos:created:success', '*').
   * @param callback The function to call when an event matching the scope is received.
   * @returns A promise that resolves to an unsubscribe function. Call this function to stop receiving events.
   */
  subscribe(
    scope: string,
    callback: (event: StoreEvent) => void,
  ): Promise<() => void>;
  /**
   * Notifies the store of an event, which can then be broadcast to subscribers.
   * @param event The StoreEvent to notify.
   * @returns A promise that resolves when the notification has been processed.
   */
  notify: (event: StoreEvent) => Promise<void>;

  /**
   * Establishes a stream of records based on provided options.
   * @param options Options for configuring the stream (e.g., batch size, delay, filters).
   * @param onStreamChange A callback function that is called when the stream's status changes.
   * @returns An object containing the async iterable stream, a cancel function, and a status getter.
   */
  stream: (
    options: TStreamOptions,
    onStreamChange: () => void,
  ) => {
    /** An async iterable that yields records as they become available in the stream. */
    stream: () => AsyncIterable<Document<T>>;
    /** A function to call to cancel the ongoing stream. */
    cancel: () => void;
    /** A getter function that returns the current status of the stream ('active', 'cancelled', or 'completed'). */
    status: () => "active" | "cancelled" | "completed";
  };

  /**
   * Creates a paginated data controller for this store.
   *
   * @param options - The pagination options, whose type is defined by the
   *                  store's `TPageOptions` generic parameter. This lets you
   *                  pass any configuration (e.g., filters, sorting, custom
   *                  pagination metadata) that your store implementation needs.
   * @returns A `PagedData<T>` controller that manages loading, navigation,
   *          resizing, and refresh.
   *
   * @example
   * // Define your store with a custom TPageOptions type
   * interface MyTodoStore extends DocumentStore<
   *   Todo,
   *   FindOptions,
   *   ReadOptions,
   *   ListOptions,
   *   { page: number; size: number; sort?: string; filter?: string }
   * > {}
   *
   * // Use it
   * const todosPaged = myTodoStore.page({
   *   page: 1,
   *   size: 10,
   *   sort: 'createdAt',
   *   filter: 'done:false'
   * });
   *
   * // Get current page state
   * const { data, loading, page } = todosPaged.page();
   * console.log(`Page ${page.number} of ${page.pages}`);
   *
   * // Navigate
   * await todosPaged.navigate(2);
   */
  page(options: TPageOptions): PagedData<T>;
}
