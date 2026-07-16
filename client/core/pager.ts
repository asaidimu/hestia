import type { QueryDSL, QueryFilter, SortConfiguration } from "@asaidimu/query";
import { DELETE_SYMBOL, type DataStore } from "@asaidimu/utils-store";
import type { Page, PagedData, PagerRefreshOptions } from "./types";
import { Debouncer } from "@asaidimu/utils-sync";

export interface PageOptions<T extends Record<string, any>> {
  page?: number;
  size?: number;
  sort?: SortConfiguration<T> | SortConfiguration<T>[];
  filter?: QueryFilter<T>;
}

const sentinel = {
  data: [],
  loading: true,
  page: {
    number: 1,
    size: 20,
    count: 0,
    total: 0,
    pages: 1,
  },
};
export function createPagedController<T extends Record<string, any>>(
  collectionName: string,
  store: DataStore<any>,
  options: PageOptions<T>,
  find: (query: QueryDSL<T>) => Promise<Page<T>>,
): PagedData<T> {
  let currentPage = options.page ?? 1;
  let currentSize = options.size ?? 20;
  let currentSort = options.sort;
  let currentFilter = options.filter;
  const debounce = new Debouncer({ delay: 50 });

  const CACHE_KEY = `${collectionName}_pager_state_`;
  store.set({
    [CACHE_KEY]: {
      data: [],
      loading: false,
      error: DELETE_SYMBOL,
      page: { number: 1, size: 20, count: 0, total: 0, pages: 1 },
    },
  });

  const buildQuery = (opts?: PagerRefreshOptions): QueryDSL<T> => {
    const page = opts?.page ?? currentPage;
    const size = opts?.size ?? currentSize;
    const query: QueryDSL<T> = {
      pagination: {
        type: "offset",
        offset: (page - 1) * size,
        limit: size,
      },
    };
    const sort = opts?.sort ?? currentSort;
    if (sort) {
      query.sort = Array.isArray(sort) ? sort : [sort];
    }
    const filter = opts?.filter ?? currentFilter;
    if (filter) {
      query.filters = filter;
    }
    return query;
  };

  const load = async (opts?: PagerRefreshOptions) => {
    await store.set({
      [CACHE_KEY]: {
        loading: true,
        error: DELETE_SYMBOL,
      },
    });

    return await new Promise((resolve) => {
      resolve(
        debounce.do(async () => {
          requestIdleCallback(() => {
            requestIdleCallback(async () => {
              try {
                const result = await find(buildQuery(opts));
                await store.set({ [CACHE_KEY]: result });
              } catch (error) {
                await store.set({
                  [CACHE_KEY]: {
                    ...sentinel,
                    error,
                  },
                });
              } finally {
                await store.set({
                  [CACHE_KEY]: {
                    loading: false,
                  },
                });
              }
            });
          });
        }),
      );
    });
  };

  const selector = store.select((s: any) => s[CACHE_KEY]);
  // TODO: Investigate bug. If by any chance a selectors subscription reaches
  // zero, it is auto discarded. This might be a problem if we are holding it
  // fix this
  const unsub = selector.subscribe(() => {});

  const controller: PagedData<T> = {
    page: () => selector.get() ?? sentinel,

    navigate: async (page: number) => {
      if (page < 1) throw new Error("Page number must be >= 1");
      currentPage = page;
      await load({
        page,
      });
    },

    resize: async (size: number, page: number) => {
      if (size < 1) throw new Error("Page size must be >= 1");
      currentSize = size;
      currentPage = page;
      await load();
    },

    sort: async (sort) => {
      currentSort = Array.isArray(sort) ? sort : [sort];
      currentPage = 1;
      await load();
    },

    filter: async (filter) => {
      currentFilter = filter;
      currentPage = 1;
      await load();
    },

    refresh: async (opts) => {
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, opts?.delay || 0)),
        load(opts),
      ]);
    },

    subscribe: (listener) => {
      return selector.subscribe(listener);
    },

    invalidate: async () => {
      await store.set({ [CACHE_KEY]: DELETE_SYMBOL });
      unsub();
    },

    id: () => CACHE_KEY,
  };
  return controller;
}
