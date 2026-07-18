import { createMatcher, createSorter } from '@asaidimu/query'; //
import type { QueryDSL, QueryFilter, SortConfiguration } from '@asaidimu/query'; //
import { DELETE_SYMBOL, ReactiveDataStore } from "@asaidimu/utils-store"; //
import type { Page, PagedData, PagerRefreshOptions } from '../core/types';

export interface FetchResult<T extends Record<string, any>> {
  data: T[];
  replace: boolean;
  scope: 'page' | 'collection';
  total?: number;
}

// Extensible parameters object embracing your DataStore infra
export interface ArrayPagerOptions<T extends Record<string, any>, F = any> {
  collectionName: string; // Unique identifier used to partition the cache matrix
  initialData?: T[];
  page?: number;
  size?: number;
  customFunctions?: Record<string, (...args: any[]) => boolean>;
  fetch?: (
    query: QueryDSL<T, F>
  ) => Promise<FetchResult<T>> | FetchResult<T>;
}

export function createArrayPager<T extends Record<string, any>, F = any>(
  options: ArrayPagerOptions<T, F>
): PagedData<T> {
  const CACHE_KEY = `${options.collectionName}_array_pager_state_`; //
  const store = new ReactiveDataStore({})

  // Baseline backing arrays managed by this adapter instance
  let pristineData = [...(options.initialData ?? [])];
  let activeScope: 'page' | 'collection' = 'collection';
  let remoteTotalRecords: number | undefined = undefined;

  // Track active filter parameters
  let currentPageNum = options.page ?? 1;
  let currentPageSize = options.size ?? 20; // Defaulting to 20 per your page controller sentinel
  let currentSort: SortConfiguration<T>[] = [];
  let currentFilter: QueryFilter<T> | undefined = undefined;

  // Tracker for errors across async horizons
  let operationalError: any = undefined;

  const { match } = createMatcher<T, F>(options.customFunctions ?? {}); //
  const { sort: localSort} = createSorter()
  const selector = store.select((s: any) => s[CACHE_KEY]); //

  // Default fallback layout conforming strictly to Page<T>
  const sentinel: Page<T> = { //
    data: [], //
    loading: true, //
    error: undefined,
    page: { //
      number: 1, //
      size: currentPageSize, //
      count: 0, //
      total: 0, //
      pages: 1, //
    },
  };

  /**
   * Constructs the current active QueryDSL footprint.
   */
  const buildQueryDSL = (): QueryDSL<T, F> => ({ //
    filters: currentFilter, //
    sort: currentSort, //
    pagination: { //
      type: 'offset', //
      offset: (currentPageNum - 1) * currentPageSize, //
      limit: currentPageSize //
    }
  });

  /**
   * Orchestrates the fetching lifecycle and syncs results into the centralized store.
   */
  const executeDataPipeline = async (isLoadingFallback = true) => {
    if (options.fetch) {
      operationalError = undefined;

      if (isLoadingFallback) {
        // Broadcast transitional loading state via the central store
        const previousState = selector.get() ?? sentinel;
        await store.set({
          [CACHE_KEY]: { ...previousState, loading: true, error: undefined }
        });
      }

      try {
        const queryDSL = buildQueryDSL();
        const result = await options.fetch(queryDSL);

        activeScope = result.scope;
        remoteTotalRecords = result.total;

        pristineData = result.replace
          ? [...result.data]
          : [...pristineData, ...result.data];
      } catch (error) {
        operationalError = error;
      }
    }

    // Process and dispatch to store
    const updatedPagePayload = computeProcessedPage(options.fetch ? isLoadingFallback : false);
    await store.set({ [CACHE_KEY]: updatedPagePayload }); //
  };

  /**
   * Coordinates local filtering, sorting and slicing transformations
   * based on the active structural scope strategy.
   */
  const computeProcessedPage = (isLoadingFlag: boolean): Page<T> => { //
    let records = [...pristineData];

    if (activeScope === 'collection') {
      // Execute standard local processing passes over the full source collection
      if (currentFilter) {
        records = records.filter((item) => match(item, currentFilter!)); //
      }
      if (currentSort.length > 0) {
        records = localSort(records, currentSort); //
      }

      const total = records.length;
      const pages = Math.ceil(total / currentPageSize) || 1;

      if (currentPageNum > pages) currentPageNum = pages;
      if (currentPageNum < 1) currentPageNum = 1;

      const startIdx = (currentPageNum - 1) * currentPageSize;
      const slicedData = records.slice(startIdx, startIdx + currentPageSize);

      return {
        data: slicedData as any,
        loading: isLoadingFlag,
        error: operationalError,
        page: { //
          number: currentPageNum,
          size: currentPageSize,
          count: slicedData.length,
          total,
          pages,
        }
      };
    }

    // Server-Driven mode: Data array has already been shaped for this target viewport
    const total = remoteTotalRecords ?? records.length;
    const pages = Math.ceil(total / currentPageSize) || 1;

    return {
      data: records as any,
      loading: isLoadingFlag,
      error: operationalError,
      page: { //
        number: currentPageNum,
        size: currentPageSize,
        count: records.length,
        total,
        pages,
      }
    };
  };

  // Kickstart immediate bootstrapping pass
  executeDataPipeline(true);

  return {
    id: () => CACHE_KEY,
    page: () => selector.get() ?? sentinel, //

    navigate: async (page: number) => {
      if (page < 1) throw new Error("Page number must be >= 1"); //
      currentPageNum = page;
      await executeDataPipeline(true);
    },

    resize: async (size: number, page: number) => {
      if (size < 1) throw new Error("Page size must be >= 1"); //
      currentPageSize = size;
      currentPageNum = page;
      await executeDataPipeline(true);
    },

    sort: async (sortConfig) => {
      currentSort = Array.isArray(sortConfig) ? sortConfig : [sortConfig];
      currentPageNum = 1; // Reset window index
      await executeDataPipeline(true);
    },

    filter: async (queryFilter) => {
      currentFilter = queryFilter;
      currentPageNum = 1; // Reset window index
      await executeDataPipeline(true);
    },

    refresh: async (opts?: PagerRefreshOptions) => { //
      const previousState = selector.get() ?? sentinel;
      await store.set({
        [CACHE_KEY]: { ...previousState, loading: true } //
      });

      const delay = opts?.delay ?? 600;

      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, delay)), //
        executeDataPipeline(false) // Handle full resolution pipeline passing down false to avoid flashing double mutations
      ]);

      const finishedState = selector.get() ?? sentinel;
      await store.set({
        [CACHE_KEY]: { ...finishedState, loading: false } //
      });
    },

    subscribe: (listener) => {
      return selector.subscribe((state) => { //
        listener(state); //
      });
    },

    invalidate: async () => {
      await store.set({ [CACHE_KEY]: DELETE_SYMBOL }); //
      pristineData = [];
    },
  };
}
