type CatalogPage<T> = {
  list: T[]
  total: number
}

export async function loadCourseCatalog<T>(
  fetchPage: (page: number, pageSize: number) => Promise<CatalogPage<T>>,
  requestedPageSize: number,
): Promise<T[]> {
  const allItems: T[] = []
  let page = 1
  let total = 0

  while (page === 1 || allItems.length < total) {
    const { list, total: pageTotal } = await fetchPage(page, requestedPageSize)
    total = pageTotal
    allItems.push(...list)

    if (list.length === 0) {
      break
    }

    page += 1
  }

  return allItems.slice(0, total)
}
